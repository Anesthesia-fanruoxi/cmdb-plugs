# Cron-Plugs Agent (本地脚本模式)

安全的任务执行Agent,只执行预置的本地脚本,由CMDB统一调度管理。

## 设计理念

**安全优先**: 不接收外部脚本内容,只执行本地预置脚本,防止任意代码执行风险。

## 项目结构

```
cron-plugs/
├── main.go              # 入口文件
├── router/              # 路由
├── model/               # 数据模型
├── common/              # 执行器和工具
└── scripts/             # 脚本目录(预置脚本)
    ├── example.sh       # 示例Shell脚本
    ├── backup_logs.sh   # 日志备份脚本
    └── test.py          # Python示例
```

## 工作流程

```
CMDB定时调度 ──调用API(脚本名)──> Agent ──执行本地脚本──> 回调CMDB上报结果
```

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 配置脚本目录

编辑 `model/const.go`:

```go
const (
    ScriptDir = "./scripts/"  // 修改为你的脚本目录
)
```

### 3. 添加脚本

将脚本放入 `scripts/` 目录:

```bash
# Linux/Mac需要添加执行权限
chmod +x scripts/*.sh
chmod +x scripts/*.py
```

### 4. 启动服务

```bash
go build -o cron-agent
./cron-agent -port 8080
```

## API使用

### 请求示例

```bash
curl -X POST http://localhost:8080/api/task/execute \
  -H "Content-Type: application/json" \
  -d '{
    "job_id": 1001,
    "project": "ops",
    "task_name": "系统信息检查",
    "task_key": "system_info",
    "script_name": "example.sh",
    "callback_url": "http://your-cmdb/api/callback",
    "timeout": 60
  }'
```

### 带参数的脚本

```bash
curl -X POST http://localhost:8080/api/task/execute \
  -H "Content-Type: application/json" \
  -d '{
    "job_id": 1002,
    "script_name": "backup_logs.sh",
    "script_args": "/var/log/app",
    "callback_url": "http://your-cmdb/api/callback"
  }'
```

### 请求参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| job_id | int64 | 是 | 任务ID |
| project | string | 否 | 项目标识 |
| task_name | string | 否 | 任务名称 |
| task_key | string | 否 | 任务标识符 |
| script_name | string | 是 | 脚本名称(相对路径) |
| script_args | string | 否 | 脚本参数 |
| callback_url | string | 是 | 回调地址 |
| timeout | int | 否 | 超时时间(秒,默认300) |

### 响应示例

```json
{
  "code": 0,
  "msg": "任务已接收,正在执行",
  "data": {
    "job_id": 1001,
    "task_name": "系统信息检查",
    "task_key": "system_info"
  }
}
```

### 回调数据

执行完成后会回调CMDB:

```json
{
  "job_id": 1001,
  "project": "ops",
  "task_name": "系统信息检查",
  "task_key": "system_info",
  "exec_status": 1,
  "result": "success",
  "error_msg": "",
  "exec_log": "脚本输出内容...",
  "start_time": "2025-10-22T18:00:00Z",
  "end_time": "2025-10-22T18:00:02Z",
  "duration": 2
}
```

## 安全特性

### 路径安全

- ✅ 防止路径穿越(`..`被拒绝)
- ✅ 只能执行scripts目录下的脚本
- ✅ 绝对路径被拒绝
- ✅ 脚本存在性验证

### 执行安全

- ✅ 脚本需预先部署和审核
- ✅ 不接收外部脚本内容
- ✅ 支持超时控制
- ✅ 标准输出和错误输出完整记录

## 支持的脚本类型

| 扩展名 | 解释器 | 示例 |
|--------|--------|------|
| .sh | bash | example.sh |
| .py | python3 | test.py |
| .ps1 | powershell | script.ps1 |
| 其他 | 直接执行 | binary |

## 脚本管理建议

### 1. 版本控制

```bash
cd scripts/
git init
git add .
git commit -m "初始化脚本库"
```

### 2. 脚本规范

```bash
#!/bin/bash
# 脚本名称: xxx
# 功能描述: xxx
# 参数说明: $1 - xxx
# 返回值: 0-成功, 1-失败

# 脚本内容...
```

### 3. 测试脚本

```bash
# 本地测试
cd scripts/
bash example.sh

# API测试
curl -X POST http://localhost:8080/api/task/execute -d '{...}'
```

### 4. 部署流程

```
1. 开发脚本 -> 本地测试
2. 提交审核 -> 代码review
3. 部署到Agent服务器 -> 重启服务
4. CMDB配置调度规则
```

## CMDB集成

### CMDB端调用示例

```go
// Go示例
func executeCronTask(jobID int64, scriptName string) error {
    data := map[string]interface{}{
        "job_id":       jobID,
        "script_name":  scriptName,
        "callback_url": "http://cmdb-server/api/cron/callback",
        "timeout":      300,
    }
    
    resp, err := http.Post("http://agent-server:8080/api/task/execute", 
        "application/json", bytes.NewBuffer(jsonData))
    // 处理响应...
}
```

### CMDB回调接口

```go
// 接收Agent回调
func callbackHandler(w http.ResponseWriter, r *http.Request) {
    var result TaskResult
    json.NewDecoder(r.Body).Decode(&result)
    
    // 更新任务状态到数据库
    updateJobStatus(result.JobID, result.ExecStatus, result.ExecLog)
}
```

## 监控和日志

### 查看Agent日志

```bash
tail -f logs/agent.log
```

### 健康检查

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

## 常见问题

### 1. 脚本不存在

**错误**: `脚本验证失败: 脚本不存在: xxx.sh`

**解决**: 检查脚本是否在scripts目录下,路径是否正确

### 2. 权限不足

**错误**: `执行失败: permission denied`

**解决**: 
```bash
chmod +x scripts/xxx.sh
```

### 3. 脚本执行超时

**错误**: `执行超时(300秒)`

**解决**: 增加timeout参数或优化脚本性能

### 4. 路径穿越拒绝

**错误**: `脚本名称不能包含'..'`

**解决**: 使用相对路径,如`backup/logs.sh`而不是`../backup/logs.sh`

## License

MIT
