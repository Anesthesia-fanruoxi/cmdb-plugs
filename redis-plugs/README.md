# Redis Plugs

轻量级 Redis 可视化管理工具后端。

## 快速开始

```bash
# 下载依赖
go mod tidy

# 运行
go run main.go
```

服务默认运行在 `http://localhost:8080`

## 配置

配置加载优先级：环境变量 > 配置文件 > 默认值

### 配置文件

编辑 `config/config.json`：

```json
{
  "server": {
    "port": ":8080"
  },
  "redis": {
    "host": "localhost",
    "port": 6379,
    "password": "",
    "db": 0
  },
  "scan": {
    "defaultCount": 1000,
    "maxCount": 10000,
    "defaultSeparator": ":"
  }
}
```

### 环境变量

```bash
SERVER_PORT=:8080
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
SCAN_MAX_COUNT=10000
SCAN_SEPARATOR=:
```

## API 接口

### 获取 Key 树

```
GET /api/tree
GET /api/tree?key=user
```

参数：
- `key`（可选）：模糊匹配关键字

### 获取 Key 值

```
GET /api/get?key=user:1001
```

参数：
- `key`（必填）：完整的 Key 名称

### 删除 Key

```
DELETE /api/delete?key=user:1001
```

参数：
- `key`（必填）：要删除的 Key 名称

## 支持的数据类型

- String
- List（最多返回 100 条）
- Set（最多返回 100 个）
- Hash（最多返回 100 个字段）
- ZSet（最多返回 100 个元素）

## 项目结构

```
redis-plugs/
├── main.go           # 程序入口
├── go.mod
├── config/
│   ├── config.go     # 配置加载
│   └── config.json   # 配置文件
├── models/
│   └── models.go     # 数据结构
├── router/
│   └── router.go     # 路由管理
├── api/
│   ├── redisTree.go  # Key 树接口
│   └── redisKey.go   # Key 操作接口
└── common/
    ├── logger.go     # 日志中间件
    ├── response.go   # 统一响应
    ├── redis.go      # Redis 连接
    ├── scanner.go    # Key 扫描
    └── data.go       # 数据操作
```
