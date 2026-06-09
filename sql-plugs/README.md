# SQL Plugs

安全的 MySQL 查询服务，支持 SQL 查询、分析、执行、检查、导出等功能。

## 快速开始

```bash
go mod tidy
go run main.go
```

服务默认运行在 `http://localhost:8090`

## 配置

配置加载优先级：环境变量 > 配置文件 > 默认值

### 配置文件

编辑 `config/config.yml`：

```yaml
server:
  port: 8090

databases:
  host: "localhost"
  port: 3306
  user: "root"
  password: "your-password"
  database: "test"
  charset: "utf8mb4"
  pool:
    max_open_conns: 25
    max_idle_conns: 10
    conn_max_lifetime: 3600
    conn_max_idle_time: 600

logs:
  level: "info"
```

## 项目结构

```
sql-plugs/
├── main.go              # 主入口
├── config/
│   ├── config.go        # 配置加载
│   └── config.yml       # 配置文件
├── common/
│   ├── database.go      # 数据库连接
│   ├── logging.go       # 日志管理
│   ├── response.go      # 统一响应
│   ├── sqlAnalyze*.go   # SQL 分析模块
│   ├── sqlComment.go    # 注释处理
│   ├── sqlSplit.go      # SQL 分割
│   └── sqlutils.go      # SQL 工具
├── model/
│   ├── request.go       # 请求模型
│   ├── response.go      # 响应模型
│   └── analyze.go       # 分析模型
├── router/
│   └── router.go        # 路由管理
├── api/
│   ├── search.go        # SQL 查询
│   ├── searchPhone.go   # 单字段查询
│   ├── searchPool.go    # 连接池状态
│   ├── searchQuery.go   # 活跃查询
│   ├── analyze.go       # SQL 分析
│   ├── execute.go       # SQL 执行（增删改）
│   ├── check.go         # SQL 检查（EXPLAIN）
│   ├── cancel.go        # 查询取消
│   ├── metadata.go      # 元数据查询
│   ├── structure.go     # 表结构查询
│   └── export.go        # 数据导出
├── static/              # SQL 分析页面静态资源
└── docs/                # 文档
```

## API 接口

### SQL 查询

**接口地址:** `POST /api/sql/search`

```json
{
  "query": "SELECT * FROM users WHERE id = 1",
  "dbName": "test"
}
```

**响应示例:**

```json
{
  "code": 200,
  "message": "查询成功",
  "data": {
    "results": [
      {
        "columns": ["id", "name", "email"],
        "rows": [[1, "张三", "zhangsan@example.com"]],
        "total": 1,
        "took": 5,
        "dbName": "test"
      }
    ]
  }
}
```

### 单字段查询

**接口地址:** `POST /api/searchPhone`

不限制返回数量，用于单字段批量查询场景。

### SQL 分析

**接口地址:** `POST /api/sql/analyze`

```json
{
  "sql": "SELECT * FROM users WHERE id = 1"
}
```

**响应示例:**

```json
{
  "code": 200,
  "message": "分析成功",
  "data": {
    "sql_type": "SELECT",
    "category": "DQL",
    "risk_level": "low",
    "features": {
      "has_where": true,
      "has_join": false,
      "has_group_by": false
    },
    "tables": ["users"],
    "columns": ["id"]
  }
}
```

### SQL 执行（增删改）

**接口地址:** `POST /api/sql/execute`

带安全限制的 DML 执行接口。

### SQL 检查

**接口地址:** `POST /api/sql/check`

EXPLAIN + 影响行数计算。

### 查询取消

**接口地址:** `POST /api/sql/cancel`

支持 KILL QUERY。

### 活跃查询列表

**接口地址:** `GET /api/sql/active`

### 连接池状态

**接口地址:** `GET /api/pool/stats`

### 元数据查询

**接口地址:** `POST /api/sql/metadata`

```json
{"type": "databases"}
{"type": "tables", "database": "test"}
{"type": "columns", "database": "test", "table": "users"}
```

### 表结构查询

**接口地址:** `POST /api/sql/structure`

```json
{
  "database": "test",
  "table": "users"
}
```

### 数据导出

**接口地址:** `POST /api/sql/export`

```json
{
  "query": "SELECT * FROM users",
  "format": "csv",
  "filename": "users_export"
}
```

### 健康检查

**接口地址:** `GET /health`

## 安全特性

- 只读查询默认只允许 SELECT/SHOW/DESCRIBE/EXPLAIN
- 风险评估：自动评估查询风险等级（低/中/高）
- 结果限制：高风险查询最多返回 100 条
- 超时控制：COUNT 查询 10 秒超时
- 查询取消：支持 KILL QUERY

## 风险等级

| 风险等级 | 条件 | 处理策略 |
|----------|------|----------|
| 低风险 | 有 LIMIT、WHERE、聚合函数 | 直接执行 |
| 中风险 | 有 JOIN、GROUP BY、DISTINCT | 直接执行 |
| 高风险 | 无任何过滤条件 | 执行 COUNT + LIMIT 100 |
