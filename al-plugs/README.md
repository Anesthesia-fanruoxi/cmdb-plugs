# AL-Plugs

一个基于 Go 语言开发的阿里云服务查询工具，提供 RESTful API 接口。

## 项目结构

```
al-plugs/
├── main.go                 # 程序入口
├── go.mod                  # 依赖管理
├── config/
│   ├── config.go          # 配置管理
│   ├── config.yaml        # 配置文件
│   └── config.example.yaml # 配置文件模板
├── model/
│   └── response.go        # 响应模型定义
├── api/
│   └── account.go         # 账户相关接口方法
├── router/
│   └── router.go          # 路由配置
└── common/
    └── client.go          # 通用工具（阿里云客户端）
```

## 功能特性

- 查询阿里云账户余额（手动查询接口）
- 定时自动检查余额（每小时一次）
- 余额低于阈值时 Webhook 告警通知
- 告警抑制机制（24小时内不重复告警）
- RESTful API 接口
- 统一的响应格式
- 完善的错误处理

## 环境要求

- Go 1.21+
- 阿里云访问凭据（AccessKey 或使用更安全的凭据方式）

## 配置说明

配置优先级：环境变量 > 配置文件 > 默认值

### 方式1: 环境变量（推荐）

```bash
# 服务端口
export PORT=8080

# 阿里云访问凭据
export ALIBABA_CLOUD_ACCESS_KEY_ID=your_access_key_id
export ALIBABA_CLOUD_ACCESS_KEY_SECRET=your_access_key_secret
export ALIBABA_CLOUD_REGION_ID=cn-hangzhou

# 告警配置
export ALERT_WEBHOOK_URL=https://your-webhook-url.com/notify
export ALERT_PROJECT="项目名称"
export ALERT_BALANCE_THRESHOLD=100.0
export ALERT_SUPPRESS_HOURS=24
export ALERT_CHECK_INTERVAL_MINUTES=60
```

### 方式2: 配置文件

复制配置文件模板：
```bash
cp config/config.example.yaml config/config.yaml
```

编辑 `config/config.yaml` 填入配置信息：
```yaml
port: "8080"
aliyun:
  access_key_id: "your_access_key_id"
  access_key_secret: "your_access_key_secret"
  region_id: "cn-hangzhou"

# 告警配置
alert:
  # Webhook 通知地址（可选）
  webhook_url: "https://your-webhook-url.com/notify"
  # 余额阈值（元），低于此值触发告警
  balance_threshold: 100.0
  # 告警抑制周期（小时），默认 24 小时
  suppress_hours: 24
  # 项目名称
  project: "项目名称"
  # 检查频次（分钟），默认 60 分钟
  check_interval_minutes: 60
```

**注意：** `config/config.yaml` 已加入 `.gitignore`，不会被提交到版本控制。

### 方式3: 阿里云凭据文件（最安全）

使用阿里云官方凭据管理方式，无需在代码或配置中存储密钥。
参考文档：https://help.aliyun.com/document_detail/378661.html

## 安装依赖

```bash
go mod tidy
```

## 运行

```bash
go run main.go
```

## API 接口

### 1. 健康检查

```
GET /health
```

响应示例：
```json
{
  "status": "ok"
}
```

### 2. 查询账户余额（手动查询）

```
GET /api/account/balance
```

响应示例：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "available_amount": "1000.00",
    "available_cash_amount": "800.00",
    "credit_amount": "200.00",
    "currency": "CNY"
  }
}
```

## 告警机制

### 定时检查
- 服务启动后立即执行一次余额检查
- 之后按配置的频次自动检查账户余额（默认每 60 分钟）

### 告警触发条件
- 当前余额 < 配置的阈值（`balance_threshold`）

### 告警抑制
- 首次触发告警时立即发送 Webhook 通知
- 记录告警时间，在抑制周期内（默认 24 小时）不会重复发送
- 超过抑制周期后，如果余额仍低于阈值，再次发送告警

### Webhook 通知格式

```json
{
  "alert_type": "balance_low",
  "alert_time": "2026-04-05 10:30:00",
  "available_amount": "50.00",
  "balance_threshold": 100.0,
  "message": "阿里云账户余额不足，当前余额: 50.00 元，阈值: 100.00 元",
  "timestamp": "2026-04-05T10:30:00Z"
}
```

## 开发计划

- [x] 账户余额查询
- [ ] ECS 实例查询
- [ ] RDS 实例查询
- [ ] OSS 存储查询
- [ ] 费用账单查询

## 许可证

MIT
