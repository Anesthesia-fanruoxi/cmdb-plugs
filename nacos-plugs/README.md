# Nacos Plugs

Nacos 配置中心的查询代理服务，支持配置获取、列表查询、搜索等功能。

## 快速开始

```bash
go mod tidy
go run main.go
```

服务默认运行在 `http://localhost:8080`

## 配置

### 环境变量

```bash
NACOS_SERVER=http://localhost:8848
NACOS_NAMESPACE=public
NACOS_USERNAME=nacos
NACOS_PASSWORD=nacos
```

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `NACOS_SERVER` | Nacos 服务地址 | `http://localhost:8848` |
| `NACOS_NAMESPACE` | 命名空间 ID | `public` |
| `NACOS_USERNAME` | 用户名 | `nacos` |
| `NACOS_PASSWORD` | 密码 | `nacos` |

## 项目结构

```
nacos-plugs/
├── main.go              # 主入口
├── config/
│   └── config.go        # 配置管理
├── common/
│   ├── client.go        # Nacos 客户端
│   └── config_service.go # 配置服务
├── model/
│   ├── config.go        # 配置模型
│   └── response.go      # 响应模型
├── router/
│   └── router.go        # 路由管理
└── api/
    └── search.go        # 查询接口
```

## API 接口

### 获取配置

**接口地址:** `GET /api/config/get?dataId=application.yml&group=DEFAULT_GROUP`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| dataId | string | 是 | 配置 ID |
| group | string | 否 | 配置分组（默认 DEFAULT_GROUP） |

**响应示例:**

```json
{
  "content": "server:\n  port: 8080\n"
}
```

### 列出配置

**接口地址:** `GET /api/config/list?pageNo=1&pageSize=10`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| pageNo | int | 否 | 页码（默认 1） |
| pageSize | int | 否 | 每页大小（默认 10） |

**响应示例:**

```json
{
  "totalCount": 100,
  "pageNumber": 1,
  "pagesAvailable": 10,
  "pageItems": [
    {
      "dataId": "application.yml",
      "group": "DEFAULT_GROUP",
      "content": "..."
    }
  ]
}
```

### 搜索配置

**接口地址:** `GET /api/config/search?dataId=application&pageNo=1&pageSize=10`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| dataId | string | 否 | 配置 ID（模糊匹配） |
| group | string | 否 | 配置分组（模糊匹配） |
| pageNo | int | 否 | 页码（默认 1） |
| pageSize | int | 否 | 每页大小（默认 10） |
