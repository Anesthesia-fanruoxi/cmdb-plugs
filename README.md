# CMDB 插件集成文档

> CMDB 平台的插件集合，每个插件独立运行，提供特定功能的 HTTP API 服务。

## 目录

- [插件概览](#插件概览)
- [插件开发规范](#插件开发规范)
- [附录](#附录)

---

## 插件概览

| 插件 | 端口 | 功能 | 详细文档 |
|------|------|------|---------|
| **es-plugs** | 8081 | Elasticsearch 查询代理 | [README](es-plugs/README.md) |
| **sql-plugs** | 8090 | MySQL 安全查询服务 | [README](sql-plugs/README.md) |
| **redis-plugs** | 8080 | Redis 可视化管理 | [README](redis-plugs/README.md) |
| **cron-plugs** | 8080 | 定时任务执行代理 | [README](cron-plugs/README.md) |
| **file-plugs** | 8083 | 文件上传管理 | [README](file-plugs/README.md) |
| **nacos-plugs** | 8080 | Nacos 配置查询 | [README](nacos-plugs/README.md) |
| **eip-plugs** | 8070 | 公网 IP 查询 | [README](eip-plugs/README.md) |
| **nginx-plugs** | 8091 | Nginx 配置管理 | [README](nginx-plugs/README.md) |
| **al-plugs** | - | 阿里云资源管理 | [README](al-plugs/README.md) |

每个插件的 API 接口文档、配置说明、使用示例均在各自目录的 `README.md` 中。

---

## 插件开发规范

创建新插件时，按照以下规范即可，无需额外描述。

### 目录结构

```
xxx-plugs/
├── main.go              # 程序入口：加载配置、初始化日志、设置路由、启动服务、优雅关闭
├── go.mod               # 独立的 Go Module
├── config/
│   ├── config.go        # 配置加载（支持配置文件 + 环境变量覆盖）
│   └── config.yml       # 配置文件
├── model/
│   ├── request.go       # 请求模型
│   └── response.go      # 响应模型
├── router/
│   └── router.go        # 路由注册 + 日志/CORS中间件
├── api/
│   └── handler.go       # 接口处理逻辑
└── common/
    ├── logging.go       # 日志初始化（使用 lumberjack + logrus）
    └── response.go      # 统一响应封装
```

### 统一响应格式

所有插件必须使用统一的 JSON 响应格式：

```json
{
  "code": 200,
  "message": "success",
  "data": { ... }
}
```

错误响应：

```json
{
  "code": 500,
  "message": "错误描述"
}
```

在 `common/response.go` 中提供 `Success(w, data)` 和 `ErrorWithCode(w, code, msg)` 方法。

### 配置规范

- 配置文件格式：YAML（`config/config.yml`）
- 配置加载优先级：**环境变量 > 配置文件 > 默认值**
- 环境变量命名：`大写_下划线`，如 `SERVER_PORT`、`DB_HOST`
- 敏感信息（密码、密钥）必须通过环境变量传入

### 路由规范

- 路由路径格式：`/api/{插件名}/{操作}`，如 `/api/nginx/generate`
- 轻量参数接口（1-2个参数）使用 **GET + query string**
- 复杂参数接口使用 **POST + JSON body**
- 删除操作使用 **DELETE + query string**
- 所有插件必须提供 `GET /health` 健康检查接口
- 中间件统一处理 CORS 和请求日志

### main.go 启动流程

```go
func main() {
    // 1. 加载配置
    // 2. 初始化日志
    // 3. 初始化业务组件（连接池、模板等）
    // 4. 设置路由
    // 5. 启动 HTTP 服务（goroutine）
    // 6. 监听系统信号，优雅关闭
}
```

### Go Module

每个插件是独立的 Go Module：

```go
module xxx-plugs

go 1.21
```

### 构建与运行

```bash
cd xxx-plugs
go mod tidy
go run main.go
```

### 检查清单

创建新插件时，确认以下项目：

- [ ] 独立 `go.mod`，模块名为 `xxx-plugs`
- [ ] `main.go` 包含优雅关闭（监听 SIGINT/SIGTERM）
- [ ] `common/response.go` 统一响应格式
- [ ] `common/logging.go` 日志初始化
- [ ] `config/` 支持配置文件 + 环境变量
- [ ] `GET /health` 健康检查接口
- [ ] CORS 中间件
- [ ] 插件目录下有 `README.md`
- [ ] 更新本文件「插件概览」表格

---

## 附录

### 配置优先级

```
环境变量 > 配置文件 > 默认值
```

### Docker 部署

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o main .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
```

### docker-compose 示例

```yaml
version: '3'
services:
  es-plugs:
    build: ./es-plugs
    ports:
      - "8081:8081"
    environment:
      - ES_HOST=http://elasticsearch:9200
      - ES_USERNAME=elastic
      - ES_PASSWORD=your-password

  sql-plugs:
    build: ./sql-plugs
    ports:
      - "8090:8090"

  redis-plugs:
    build: ./redis-plugs
    ports:
      - "8080:8080"
    environment:
      - REDIS_HOST=redis
      - REDIS_PORT=6379
```

### 安全建议

1. 生产环境使用环境变量管理敏感信息
2. 定期 `go mod tidy` 更新依赖
3. 生产环境建议通过反向代理启用 HTTPS
4. 确保日志中不包含敏感信息
