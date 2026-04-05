package api

import (
	"encoding/json"
	"strings"
	"time"

	"al-plugs/common"
	"al-plugs/config"
	"al-plugs/logger"
	"al-plugs/model"

	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
)

// AccountAPI 账户相关接口
type AccountAPI struct {
	cfg *config.Config
}

// NewAccountAPI 创建账户API实例
func NewAccountAPI(cfg *config.Config) *AccountAPI {
	return &AccountAPI{
		cfg: cfg,
	}
}

// QueryBalance 查询账户余额
func (a *AccountAPI) QueryBalance(c *gin.Context) {
	startTime := time.Now()

	// 创建客户端
	client, err := common.CreateBssClient(a.cfg)
	if err != nil {
		logger.Error("创建客户端失败: %v", err)
		c.JSON(500, model.NewErrorResponse(500, "创建客户端失败: "+err.Error()))
		return
	}

	// 调用API
	runtime := &util.RuntimeOptions{}
	resp, err := client.QueryAccountBalanceWithOptions(runtime)
	if err != nil {
		// 处理SDK错误
		if sdkErr, ok := err.(*tea.SDKError); ok {
			errorMsg := tea.StringValue(sdkErr.Message)

			// 解析诊断信息
			var data interface{}
			d := json.NewDecoder(strings.NewReader(tea.StringValue(sdkErr.Data)))
			d.Decode(&data)

			if m, ok := data.(map[string]interface{}); ok {
				if recommend, exists := m["Recommend"]; exists {
					errorMsg += " | 建议: " + recommend.(string)
				}
			}

			logger.Error("查询账户余额失败: %s", errorMsg)
			c.JSON(500, model.NewErrorResponse(500, errorMsg))
			return
		}
		logger.Error("查询账户余额失败: %v", err)
		c.JSON(500, model.NewErrorResponse(500, err.Error()))
		return
	}

	// 构造响应数据
	balance := &model.AccountBalance{}
	if resp.Body != nil && resp.Body.Data != nil {
		balance.AvailableAmount = tea.StringValue(resp.Body.Data.AvailableAmount)
		balance.AvailableCashAmount = tea.StringValue(resp.Body.Data.AvailableCashAmount)
		balance.CreditAmount = tea.StringValue(resp.Body.Data.CreditAmount)
		balance.Currency = tea.StringValue(resp.Body.Data.Currency)
	}

	elapsed := time.Since(startTime).Milliseconds()
	logger.Infof("查询账户余额成功", "总耗时: %dms", elapsed)

	c.JSON(200, model.NewSuccessResponse(balance))
}
