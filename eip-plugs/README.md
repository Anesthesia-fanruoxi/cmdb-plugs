# EIP Plugs

公网 IP 查询服务，通过多个公共 API 并发查询，确保结果准确性。

## 快速开始

```bash
go mod tidy
go run main.go
```

服务默认运行在 `http://localhost:8070`

## 项目结构

```
eip-plugs/
├── main.go              # 主入口
├── common/
│   ├── logger.go        # 日志管理
│   └── response.go      # 统一响应
├── model/
│   └── ip_response.go   # IP 模型
├── router/
│   └── router.go        # 路由管理
└── api/
    └── ip_handler.go    # IP 查询接口
```

## API 接口

### 获取公网 IP

**接口地址:** `GET /api/ip`

**请求示例:**

```bash
curl http://localhost:8070/api/ip
```

**响应示例:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "ip": "123.45.67.89"
  }
}
```

## 工作原理

服务并发查询以下公共 IP 服务，返回最快响应的结果：

1. `https://ident.me`
2. `https://ipv4.icanhazip.com`
3. `http://myip.ipip.net/ip`

## 使用场景

- 服务器公网 IP 自动上报
- 网络环境检测
- IP 地址变更监控
- CMDB 资产信息自动更新
