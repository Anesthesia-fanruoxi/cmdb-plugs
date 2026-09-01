package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sql-plugs/common"
	"time"

	"github.com/xuri/excelize/v2"
)

const (
	exportQueryTimeout = 30 * time.Minute // 导出查询超时
	maxRowsPerSheet    = 1000000          // 单个Sheet最大行数，超过后自动分Sheet
)

// ExportAsyncHandler 异步导出入口，立即返回task_id
func ExportAsyncHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.ErrorWithCode(w, http.StatusMethodNotAllowed, "只允许POST请求")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		common.ErrorWithCode(w, http.StatusBadRequest, "读取请求体失败: "+err.Error())
		return
	}
	defer r.Body.Close()

	var rawReq struct {
		Query       json.RawMessage `json:"query"`
		DB          string          `json:"db_name"`
		CallbackURL string          `json:"callback_url"`
		UploadURL   string          `json:"upload_url"`
		TaskID      string          `json:"task_id"`
	}
	if err := json.Unmarshal(body, &rawReq); err != nil {
		common.ErrorWithCode(w, http.StatusBadRequest, "请求参数解析失败: "+err.Error())
		return
	}

	var queryStr string
	if err := json.Unmarshal(rawReq.Query, &queryStr); err != nil {
		common.ErrorWithCode(w, http.StatusBadRequest, "query字段解析失败: "+err.Error())
		return
	}

	if queryStr == "" {
		common.ErrorWithCode(w, http.StatusBadRequest, "查询语句不能为空")
		return
	}
	if rawReq.CallbackURL == "" {
		common.ErrorWithCode(w, http.StatusBadRequest, "callback_url不能为空")
		return
	}
	if rawReq.TaskID == "" {
		common.ErrorWithCode(w, http.StatusBadRequest, "task_id不能为空")
		return
	}
	if rawReq.UploadURL == "" {
		common.ErrorWithCode(w, http.StatusBadRequest, "upload_url不能为空")
		return
	}

	if err := validateExportQuery(queryStr); err != nil {
		common.ErrorWithCode(w, http.StatusBadRequest, err.Error())
		return
	}

	taskID := rawReq.TaskID
	filePath := filepath.Join("tmp", fmt.Sprintf("sql_export_%s.xlsx", taskID))

	common.Logger.Infof("异步导出任务已提交 - taskID: %s, 数据库: %s, SQL:\n%s", taskID, rawReq.DB, queryStr)

	tm := common.GetTaskManager()
	if err := tm.Submit(func() {
		runExportTask(taskID, rawReq.DB, queryStr, rawReq.CallbackURL, rawReq.UploadURL, filePath)
	}); err != nil {
		common.ErrorWithCode(w, http.StatusTooManyRequests, err.Error())
		return
	}

	common.SuccessWithMessage(w, "导出任务已创建", map[string]string{
		"task_id": taskID,
	})
}

// runExportTask 后台执行导出任务（分步回调）
func runExportTask(taskID, dbName, query, callbackURL, uploadURL, filePath string) {
	state := NewExportTaskState(taskID, callbackURL, dbName)
	common.Logger.Infof("导出任务开始 - taskID: %s", taskID)

	os.MkdirAll("tmp", 0755)
	defer os.Remove(filePath)

	// Step 1: 初始化（创建DB连接，直接连接目标库）
	state.StartStep("setup1")

	if dbName != "" && !common.IsValidDatabaseName(dbName, 64) {
		state.FailTask("无效的数据库名称: " + dbName)
		return
	}

	exportDB, err := common.CreateExportDB(dbName)
	if err != nil {
		common.Logger.Errorf("导出任务[%s] 创建DB连接失败: %v", taskID, err)
		state.FailTask("创建DB连接失败: " + err.Error())
		return
	}
	defer exportDB.Close()

	state.CompleteStep("setup1")
	state.SendCallback()

	// Step 2: 流式查询+写入xlsx
	state.StartStep("setup2")
	common.Logger.Infof("导出任务[%s] setup2开始 - 执行流式查询", taskID)
	rowCount, err := streamQueryToXLSX(exportDB, query, filePath, state)
	if err != nil {
		common.Logger.Errorf("导出任务[%s] 写入xlsx失败: %v", taskID, err)
		state.FailTask("写入xlsx失败: " + err.Error())
		return
	}
	state.CompleteStep("setup2")
	state.payload.TotalCount = rowCount
	common.Logger.Infof("导出任务[%s] setup2完成 - 行数: %d, 文件: %s", taskID, rowCount, filePath)

	// Step 3: 上传文件
	state.StartStep("setup3")
	state.SendCallback()
	common.Logger.Infof("导出任务[%s] setup3开始 - 上传文件到: %s", taskID, uploadURL)

	fileID, err := uploadToCMDB(uploadURL, filePath)
	if err != nil {
		common.Logger.Errorf("导出任务[%s] 上传失败: %v", taskID, err)
		state.FailTask("上传文件失败: " + err.Error())
		return
	}
	state.CompleteStep("setup3")
	common.Logger.Infof("导出任务[%s] 上传成功 - fileID: %s", taskID, fileID)

	// Step 4: 完成通知
	state.StartStep("setup4")
	state.CompleteStep("setup4")
	state.SucceedTask(fileID)
	common.Logger.Infof("导出任务[%s] 完成", taskID)
}

// streamQueryToXLSX 流式查询并写入xlsx
func streamQueryToXLSX(db *sql.DB, query, filePath string, state *ExportTaskState) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), exportQueryTimeout)
	defer cancel()

	taskID := state.payload.ID

	// MySQL侧超时双保险（毫秒），与Go context超时保持一致
	timeoutMS := uint64(exportQueryTimeout / time.Millisecond)
	db.Exec(fmt.Sprintf("SET SESSION max_execution_time = %d", timeoutMS))

	common.Logger.Infof("导出任务[%s] 开始执行查询...", taskID)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			common.Logger.Errorf("导出任务[%s] 查询超时(>%v), 连接已断开", taskID, exportQueryTimeout)
			db.Close() // 强制关闭连接，确保MySQL会话释放
			return 0, fmt.Errorf("查询超时（超过%v），已强制断开连接", exportQueryTimeout)
		}
		common.Logger.Errorf("导出任务[%s] 查询执行失败: %v", taskID, err)
		return 0, fmt.Errorf("执行查询失败: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return 0, fmt.Errorf("获取列名失败: %w", err)
	}
	common.Logger.Infof("导出任务[%s] 查询返回 %d 列: %v", taskID, len(columns), columns)

	// 构建表头
	headerRow := make([]interface{}, len(columns))
	for i, col := range columns {
		headerRow[i] = col
	}

	f := excelize.NewFile()
	defer f.Close()

	// 初始化第一个 Sheet
	sheetIndex := 1
	sheetName := "Sheet1"
	sw, err := f.NewStreamWriter(sheetName)
	if err != nil {
		return 0, fmt.Errorf("创建StreamWriter失败: %w", err)
	}
	if err := sw.SetRow("A1", headerRow); err != nil {
		return 0, fmt.Errorf("写入表头失败: %w", err)
	}

	totalRows := 0
	sheetRowCount := 0 // 当前 Sheet 已写入行数（不含表头）
	rowNum := 2        // 当前 Sheet 内的行号（表头占第1行）

	common.Logger.Infof("导出任务[%s] 开始遍历数据行...", taskID)
	values := make([]sql.RawBytes, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		excelRow := make([]interface{}, len(columns))
		for i, val := range values {
			if val == nil {
				excelRow[i] = ""
			} else {
				excelRow[i] = string(val)
			}
		}

		// 检查是否需要分 Sheet
		if sheetRowCount >= maxRowsPerSheet {
			common.Logger.Infof("导出任务[%s] Sheet%d 已满 %d 行，切到下一个Sheet", taskID, sheetIndex, sheetRowCount)
			if err := sw.Flush(); err != nil {
				common.Logger.Warnf("导出任务[%s] Sheet%d Flush失败: %v", taskID, sheetIndex, err)
			}

			sheetIndex++
			sheetName = fmt.Sprintf("Sheet%d", sheetIndex)
			f.NewSheet(sheetName)
			sw, err = f.NewStreamWriter(sheetName)
			if err != nil {
				return totalRows, fmt.Errorf("创建Sheet%d StreamWriter失败: %w", sheetIndex, err)
			}
			if err := sw.SetRow("A1", headerRow); err != nil {
				return totalRows, fmt.Errorf("Sheet%d 写入表头失败: %w", sheetIndex, err)
			}

			sheetRowCount = 0
			rowNum = 2
			common.Logger.Infof("导出任务[%s] 已创建 %s", taskID, sheetName)
		}

		cell, _ := excelize.CoordinatesToCellName(1, rowNum)
		sw.SetRow(cell, excelRow)
		totalRows++
		sheetRowCount++
		rowNum++
	}

	if err := rows.Err(); err != nil {
		common.Logger.Errorf("导出任务[%s] 遍历数据异常: %v (已处理%d行)", taskID, err, totalRows)
		return totalRows, fmt.Errorf("遍历数据失败: %w", err)
	}

	// Flush 最后一个 Sheet
	common.Logger.Infof("导出任务[%s] 数据遍历完成(%d行, %d个Sheet), 最终Flush并保存文件...", taskID, totalRows, sheetIndex)
	if err := sw.Flush(); err != nil {
		return totalRows, fmt.Errorf("最终Flush失败: %w", err)
	}

	if err := f.SaveAs(filePath); err != nil {
		return totalRows, fmt.Errorf("保存xlsx失败: %w", err)
	}
	common.Logger.Infof("导出任务[%s] 文件保存成功: %s", taskID, filePath)

	return totalRows, nil
}
