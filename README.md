# CMDB 插件集成文档

> 本文档详细说明了所有 CMDB 插件的功能、配置和使用方法

## 📋 目录

- [插件概览](#插件概览)
- [1. ES-Plugs - Elasticsearch查询服务](#1-es-plugs---elasticsearch查询服务)
- [2. SQL-Plugs - SQL查询服务](#2-sql-plugs---sql查询服务)
- [3. Redis-Plugs - Redis可视化管理](#3-redis-plugs---redis可视化管理)
- [4. Cron-Plugs - 定时任务执行代理](#4-cron-plugs---定时任务执行代理)
- [5. File-Plugs - 文件上传服务](#5-file-plugs---文件上传服务)
- [6. Nacos-Plugs - Nacos配置管理](#6-nacos-plugs---nacos配置管理)
- [7. EIP-Plugs - 公网IP查询](#7-eip-plugs---公网ip查询)

---

## 插件概览

| 插件名称 | 端口 | 主要功能 | 技术栈 |
|---------|------|---------|--------|
| **es-plugs** | 8081 | Elasticsearch查询代理 | Go + ES HTTP API |
| **sql-plugs** | 8082 | MySQL安全查询服务 | Go + MySQL |
| **redis-plugs** | 8080 | Redis可视化管理 | Go + Redis |
| **cron-plugs** | 8080 | 定时任务执行代理 | Go + Shell/Python |
| **file-plugs** | 8083 | 文件上传管理 | Go + 文件系统 |
| **nacos-plugs** | 8080 | Nacos配置查询 | Go + Nacos API |
| **eip-plugs** | 8070 | 公网IP查询 | Go + HTTP |

---

## 1. ES-Plugs - Elasticsearch查询服务

### 功能概述

提供安全、智能的Elasticsearch查询代理服务，支持Kibana风格语法、滚动查询、上下文查询等功能。

### 核心特性

- ✅ **Kibana风格查询** - 支持Kibana语法，智能转换为ES查询
- ✅ **时间排序查询** - 支持按时间升序/降序排序
- ✅ **索引映射查询** - 获取索引列表和字段映射
- ✅ **滚动查询** - 大数据量遍历，支持init/continue/clear操作
- ✅ **上下文查询** - 获取指定文档前后的相关记录
- ✅ **环境变量支持** - 配置可通过环境变量覆盖
- ✅ **标准HTTP请求** - 无第三方SDK依赖

### 项目结构

```
es-plugs/
├── main.go              # 主入口
├── .env.example         # 环境变量配置模板
├── config/
│   ├── config.go       # 配置加载（支持环境变量）
│   └── config.yml      # 配置文件
├── common/
│   ├── logging.go      # 自定义日志
│   └── response.go     # 统一响应
├── model/
│   ├── model.go        # 数据模型
│   ├── scroll.go       # 滚动查询模型
│   └── context.go      # 上下文查询模型
├── router/
│   └── router.go       # 路由管理
└── api/
    ├── search.go       # Kibana风格查询
    ├── indices.go      # 索引映射查询
    ├── scroll.go       # 滚动查询
    └── context.go      # 上下文查询
```

### 快速开始

#### 1. 配置ES连接

**方式一：配置文件（开发环境）**

编辑 `config/config.yml`:

```yaml
elasticsearch:
  host: "http://localhost:9200"
  username: "elastic"
  password: "your-password"
  timeout: 30

log:
  level: "info"  # info 或 error
```

**方式二：环境变量（生产环境推荐）**

```bash
export ES_HOST="http://localhost:9200"
export ES_USERNAME="elastic"
export ES_PASSWORD="your-password"
export LOG_LEVEL="info"
export LIMIT_MAX_SIZE="3000"
```

#### 2. 启动服务

```bash
go mod tidy
go run main.go
```

服务将在 `http://localhost:8081` 启动

### API接口

#### 查询ES（增强版）

**接口地址:** `POST /api/es/query`

**请求示例:**

```json
{
  "index": "my-index",
  "query": {
    "match": {
      "field_name": "search_value"
    }
  },
  "size": 20,
  "from": 0,
  "sort": [
    {
      "created_at": {
        "order": "desc"
      }
    }
  ]
}
```

#### 获取索引列表

**接口地址:** `POST /api/es/indices`

**请求示例:**

```json
{
  "index_pattern": "my-index-*"
}
```

#### 滚动查询

**接口地址:** `POST /api/es/scroll`

**初始化滚动查询:**

```json
{
  "action": "init",
  "index": "jxh_sms_*",
  "start_time": "2025-10-17 11:03:11",
  "end_time": "2025-10-17 11:33:11",
  "time_field": "sendTimeStamp",
  "keyword": "中国移动 or 中国联通",
  "size": 2000,
  "scroll_time": "1m"
}
```

**继续滚动:**

```json
{
  "action": "continue",
  "scroll_id": "DXF1ZXJ5QW5kRmV0Y2g...",
  "scroll_time": "1m"
}
```

**清除滚动:**

```json
{
  "action": "clear",
  "scroll_id": "DXF1ZXJ5QW5kRmV0Y2g..."
}
```

#### 上下文查询

**接口地址:** `POST /api/es/context`

**请求示例:**

```json
{
  "index": "my-index",
  "doc_id": "center-doc-id",
  "before": 10,
  "after": 10,
  "sort_field": "@timestamp"
}
```

### 配置说明

**配置优先级:** 环境变量 > 配置文件 > 默认值

**环境变量列表:**

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `ES_HOST` | ES服务地址 | `http://localhost:9200` |
| `ES_USERNAME` | ES用户名 | `elastic` |
| `ES_PASSWORD` | ES密码 | `` |
| `LOG_LEVEL` | 日志级别 | `info` |
| `LIMIT_MAX_SIZE` | 最大返回条数 | `3000` |

---

## 2. SQL-Plugs - SQL查询服务

### 功能概述

提供安全、智能的MySQL查询服务，支持SQL解析、风险评估、查询限制等功能。

### 核心特性

- ✅ **智能SQL解析** - 自动识别SQL类型和风险等级
- ✅ **安全限制** - 只允许DQL查询，拒绝DML/DDL操作
- ✅ **风险评估** - 根据SQL特性评估查询风险
- ✅ **查询限制** - 最多返回100条数据，但返回真实总数
- ✅ **批量查询** - 支持多条SQL语句（分号分隔）
- ✅ **查询取消** - 支持KILL QUERY
- ✅ **元数据查询** - 获取数据库、表、字段信息
- ✅ **数据导出** - 支持CSV/Excel导出

### 项目结构

```
sql-plugs/
├── main.go              # 主入口
├── config/
│   ├── config.go       # 配置加载
│   └── config.yml      # 配置文件
├── common/
│   ├── database.go     # 数据库连接
│   ├── logging.go      # 日志管理
│   ├── sqlAnalyze*.go  # SQL分析模块
│   ├── sqlComment.go   # 注释处理
│   ├── sqlSplit.go     # SQL分割
│   └── sqlutils.go     # SQL工具
├── model/
│   ├── request.go      # 请求模型
│   ├── response.go     # 响应模型
│   └── analyze.go      # 分析模型
├── router/
│   └── router.go       # 路由管理
└── api/
    ├── search.go       # SQL查询
    ├── analyze.go      # SQL分析
    ├── metadata.go     # 元数据查询
    ├── structure.go    # 表结构查询
    ├── export.go       # 数据导出
    └── cancel.go       # 查询取消
```

### 快速开始

#### 1. 配置数据库连接

编辑 `config/config.yml`:

```yaml
database:
  host: "localhost"
  port: 3306
  username: "root"
  password: "your-password"
  database: "test"
  max_open_conns: 10
  max_idle_conns: 5

server:
  port: 8082

log:
  level: "info"
```

#### 2. 启动服务

```bash
go mod tidy
go run main.go
```

服务将在 `http://localhost:8082` 启动

### API接口

#### SQL查询

**接口地址:** `POST /api/sql/search`

**请求示例:**

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
        "rows": [
          [1, "张三", "zhangsan@example.com"]
        ],
        "total": 1,
        "took": 5,
        "dbName": "test"
      }
    ],
    "total": 1,
    "took": 5
  }
}
```

#### SQL分析

**接口地址:** `POST /api/sql/analyze`

**请求示例:**

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

#### 元数据查询

**接口地址:** `POST /api/sql/metadata`

**获取数据库列表:**

```json
{
  "type": "databases"
}
```

**获取表列表:**

```json
{
  "type": "tables",
  "database": "test"
}
```

**获取字段列表:**

```json
{
  "type": "columns",
  "database": "test",
  "table": "users"
}
```

#### 表结构查询

**接口地址:** `POST /api/sql/structure`

**请求示例:**

```json
{
  "database": "test",
  "table": "users"
}
```

#### 数据导出

**接口地址:** `POST /api/sql/export`

**请求示例:**

```json
{
  "query": "SELECT * FROM users",
  "format": "csv",
  "filename": "users_export"
}
```

### 查询逻辑说明

#### 风险等级评估

| 风险等级 | 条件 | 处理策略 |
|----------|------|----------|
| **低风险** | 有LIMIT、WHERE、聚合函数 | 直接执行，遍历获取总数 |
| **中风险** | 有JOIN、GROUP BY、DISTINCT | 直接执行，遍历获取总数 |
| **高风险** | 无任何过滤条件 | 执行COUNT，强制LIMIT 100 |

#### 查询流程

```
用户输入SQL
    ↓
1. SQL格式化（规范化空白字符）
    ↓
2. 获取SQL类型，确保是DQL
    ↓
3. 判断风险程度
    ↓
4. 根据风险等级处理
    ├─ 高风险 ──→ 执行COUNT ──→ 添加LIMIT 100 ──→ 执行查询
    └─ 低/中风险 ──→ 直接执行 ──→ 遍历全部结果
    ↓
5. 返回最多100条数据 + 真实总数
```

### 安全特性

- ✅ **只读限制** - 只允许SELECT/SHOW/DESCRIBE/EXPLAIN
- ✅ **风险评估** - 自动评估查询风险
- ✅ **结果限制** - 最多返回100条数据
- ✅ **超时控制** - COUNT查询10秒超时
- ✅ **查询取消** - 支持KILL QUERY

---

## 3. Redis-Plugs - Redis可视化管理

### 功能概述

轻量级Redis可视化管理工具后端，提供Key树形展示、数据查询、删除等功能。

### 核心特性

- ✅ **Key树形展示** - 按分隔符自动构建树形结构
- ✅ **模糊搜索** - 支持Key模糊匹配
- ✅ **多类型支持** - String/List/Set/Hash/ZSet
- ✅ **环境变量配置** - 支持环境变量覆盖
- ✅ **连接池管理** - 自动管理Redis连接

### 项目结构

```
redis-plugs/
├── main.go           # 程序入口
├── config/
│   ├── config.go     # 配置加载
│   └── config.json   # 配置文件
├── models/
│   └── models.go     # 数据结构
├── router/
│   └── router.go     # 路由管理
├── api/
│   ├── redisTree.go  # Key树接口
│   ├── redisKey.go   # Key操作接口
│   └── redisInfo.go  # Redis信息接口
└── common/
    ├── logger.go     # 日志中间件
    ├── response.go   # 统一响应
    ├── redis.go      # Redis连接
    ├── scanner.go    # Key扫描
    └── data.go       # 数据操作
```

### 快速开始

#### 1. 配置Redis连接

**方式一：配置文件**

编辑 `config/config.json`:

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

**方式二：环境变量**

```bash
export SERVER_PORT=":8080"
export REDIS_HOST="localhost"
export REDIS_PORT="6379"
export REDIS_PASSWORD=""
export REDIS_DB="0"
export SCAN_MAX_COUNT="10000"
export SCAN_SEPARATOR=":"
```

#### 2. 启动服务

```bash
go mod tidy
go run main.go
```

服务将在 `http://localhost:8080` 启动

### API接口

#### 获取Key树

**接口地址:** `GET /api/tree`

**请求示例:**

```bash
# 获取所有Key树
curl http://localhost:8080/api/tree

# 模糊搜索
curl http://localhost:8080/api/tree?key=user
```

**响应示例:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "tree": [
      {
        "name": "user",
        "type": "folder",
        "children": [
          {
            "name": "1001",
            "type": "string",
            "key": "user:1001"
          }
        ]
      }
    ],
    "total": 1
  }
}
```

#### 获取Key值

**接口地址:** `GET /api/get`

**请求示例:**

```bash
curl "http://localhost:8080/api/get?key=user:1001"
```

**响应示例:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "key": "user:1001",
    "type": "string",
    "value": "张三",
    "ttl": -1
  }
}
```

#### 删除Key

**接口地址:** `DELETE /api/delete`

**请求示例:**

```bash
curl -X DELETE "http://localhost:8080/api/delete?key=user:1001"
```

### 支持的数据类型

| 类型 | 说明 | 限制 |
|------|------|------|
| String | 字符串 | 完整返回 |
| List | 列表 | 最多返回100条 |
| Set | 集合 | 最多返回100个 |
| Hash | 哈希表 | 最多返回100个字段 |
| ZSet | 有序集合 | 最多返回100个元素 |

### 配置说明

**配置优先级:** 环境变量 > 配置文件 > 默认值

---

## 4. Cron-Plugs - 定时任务执行代理

### 功能概述

安全的任务执行Agent，只执行预置的本地脚本，由CMDB统一调度管理。

### 核心特性

- ✅ **安全优先** - 不接收外部脚本内容，只执行本地预置脚本
- ✅ **路径安全** - 防止路径穿越，只能执行scripts目录下的脚本
- ✅ **超时控制** - 支持脚本执行超时设置
- ✅ **异步回调** - 执行完成后回调CMDB上报结果
- ✅ **多脚本支持** - 支持Shell、Python、PowerShell等

### 项目结构

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

### 工作流程

```
CMDB定时调度 ──调用API(脚本名)──> Agent ──执行本地脚本──> 回调CMDB上报结果
```

### 快速开始

#### 1. 配置脚本目录

编辑 `model/const.go`:

```go
const (
    ScriptDir = "./scripts/"  // 修改为你的脚本目录
)
```

#### 2. 添加脚本

将脚本放入 `scripts/` 目录:

```bash
# Linux/Mac需要添加执行权限
chmod +x scripts/*.sh
chmod +x scripts/*.py
```

#### 3. 启动服务

```bash
go mod tidy
go build -o cron-agent
./cron-agent -port 8080
```

### API接口

#### 执行任务

**接口地址:** `POST /api/task/execute`

**请求示例:**

```json
{
  "job_id": 1001,
  "project": "ops",
  "task_name": "系统信息检查",
  "task_key": "system_info",
  "script_name": "example.sh",
  "callback_url": "http://your-cmdb/api/callback",
  "timeout": 60
}
```

**带参数的脚本:**

```json
{
  "job_id": 1002,
  "script_name": "backup_logs.sh",
  "script_args": "/var/log/app",
  "callback_url": "http://your-cmdb/api/callback"
}
```

**请求参数说明:**

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

**响应示例:**

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

**回调数据:**

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

### 安全特性

#### 路径安全

- ✅ 防止路径穿越(`..`被拒绝)
- ✅ 只能执行scripts目录下的脚本
- ✅ 绝对路径被拒绝
- ✅ 脚本存在性验证

#### 执行安全

- ✅ 脚本需预先部署和审核
- ✅ 不接收外部脚本内容
- ✅ 支持超时控制
- ✅ 标准输出和错误输出完整记录

### 支持的脚本类型

| 扩展名 | 解释器 | 示例 |
|--------|--------|------|
| .sh | bash | example.sh |
| .py | python3 | test.py |
| .ps1 | powershell | script.ps1 |
| 其他 | 直接执行 | binary |

### 脚本管理建议

#### 1. 版本控制

```bash
cd scripts/
git init
git add .
git commit -m "初始化脚本库"
```

#### 2. 脚本规范

```bash
#!/bin/bash
# 脚本名称: xxx
# 功能描述: xxx
# 参数说明: $1 - xxx
# 返回值: 0-成功, 1-失败

# 脚本内容...
```

#### 3. 部署流程

```
1. 开发脚本 -> 本地测试
2. 提交审核 -> 代码review
3. 部署到Agent服务器 -> 重启服务
4. CMDB配置调度规则
```

---

## 5. File-Plugs - 文件上传服务

### 功能概述

提供安全的文件上传管理服务，支持多文件上传、目录结构保留、自动解压等功能。

### 核心特性

- ✅ **多文件上传** - 支持批量文件上传
- ✅ **目录结构保留** - 保留原始目录结构
- ✅ **自动解压** - 支持ZIP/RAR自动解压
- ✅ **文件类型限制** - 可配置允许的文件类型
- ✅ **大小限制** - 可配置最大文件大小
- ✅ **多路径配置** - 支持多个上传路径配置

### 项目结构

```
file-plugs/
├── main.go              # 主入口
├── config/
│   ├── config.go       # 配置加载
│   └── config.yaml     # 配置文件
├── common/
│   ├── storage.go      # 文件存储
│   ├── unzip.go        # 解压处理
│   └── validate.go     # 文件验证
└── api/
    ├── upload.go       # 文件上传
    ├── list.go         # 文件列表
    └── keys.go         # 路径列表
```

### 快速开始

#### 1. 配置上传路径

编辑 `config/config.yaml`:

```yaml
server:
  port: 8083

storage:
  max_file_size: 10485760  # 10MB
  paths:
    - key: "logs"
      path: "./uploads/logs"
      max_file_size: 5242880  # 5MB
      allowed_types: [".log", ".txt"]
      auto_unzip: false
    
    - key: "images"
      path: "./uploads/images"
      allowed_types: [".jpg", ".png", ".gif"]
      auto_unzip: false
    
    - key: "packages"
      path: "./uploads/packages"
      allowed_types: [".zip", ".rar", ".tar.gz"]
      auto_unzip: true
```

#### 2. 启动服务

```bash
go mod tidy
go run main.go
```

服务将在 `http://localhost:8083` 启动

### API接口

#### 文件上传

**接口地址:** `POST /api/upload`

**请求参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| key | string | 是 | 上传路径标识 |
| path | string | 否 | 子目录路径 |
| file | file | 是 | 上传的文件 |
| filePath | string | 否 | 文件相对路径（保留目录结构） |

**单文件上传示例:**

```bash
curl -X POST http://localhost:8083/api/upload \
  -F "key=logs" \
  -F "file=@app.log"
```

**多文件上传示例:**

```bash
curl -X POST http://localhost:8083/api/upload \
  -F "key=images" \
  -F "file=@photo1.jpg" \
  -F "file=@photo2.jpg"
```

**保留目录结构上传:**

```bash
curl -X POST http://localhost:8083/api/upload \
  -F "key=logs" \
  -F "file=@app.log" \
  -F "filePath=2025/01/19/app.log"
```

**响应示例:**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "key": "logs",
    "files": [
      "2025/01/19/app.log"
    ],
    "count": 1,
    "size": 1024,
    "errors": []
  }
}
```

#### 获取文件列表

**接口地址:** `GET /api/list`

**请求示例:**

```bash
curl "http://localhost:8083/api/list?key=logs"
```

**响应示例:**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "files": [
      {
        "name": "app.log",
        "path": "2025/01/19/app.log",
        "size": 1024,
        "modified": "2025-01-19T10:00:00Z"
      }
    ],
    "total": 1
  }
}
```

#### 获取路径列表

**接口地址:** `GET /api/keys`

**响应示例:**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "keys": [
      {
        "key": "logs",
        "path": "./uploads/logs",
        "allowed_types": [".log", ".txt"]
      },
      {
        "key": "images",
        "path": "./uploads/images",
        "allowed_types": [".jpg", ".png", ".gif"]
      }
    ]
  }
}
```

### 配置说明

#### 路径配置

| 字段 | 类型 | 说明 |
|------|------|------|
| key | string | 路径标识（唯一） |
| path | string | 实际存储路径 |
| max_file_size | int64 | 最大文件大小（字节） |
| allowed_types | []string | 允许的文件类型 |
| auto_unzip | bool | 是否自动解压 |

#### 安全特性

- ✅ **路径验证** - 禁止根目录和当前目录
- ✅ **类型限制** - 可配置允许的文件类型
- ✅ **大小限制** - 可配置最大文件大小
- ✅ **路径穿越防护** - 防止路径穿越攻击

---

## 6. Nacos-Plugs - Nacos配置管理

### 功能概述

提供Nacos配置中心的查询代理服务，支持配置获取、列表查询、搜索等功能。

### 核心特性

- ✅ **配置查询** - 获取指定配置内容
- ✅ **配置列表** - 分页查询配置列表
- ✅ **配置搜索** - 按dataId和group搜索
- ✅ **统一代理** - 统一的Nacos访问入口

### 项目结构

```
nacos-plugs/
├── main.go              # 主入口
├── config/
│   └── config.go       # 配置管理
├── common/
│   ├── client.go       # Nacos客户端
│   └── config_service.go  # 配置服务
├── model/
│   ├── config.go       # 配置模型
│   └── response.go     # 响应模型
├── router/
│   └── router.go       # 路由管理
└── api/
    └── search.go       # 查询接口
```

### 快速开始

#### 1. 配置Nacos连接

编辑配置或使用环境变量:

```bash
export NACOS_SERVER="http://localhost:8848"
export NACOS_NAMESPACE="public"
export NACOS_USERNAME="nacos"
export NACOS_PASSWORD="nacos"
```

#### 2. 启动服务

```bash
go mod tidy
go run main.go
```

服务将在 `http://localhost:8080` 启动

### API接口

#### 获取配置

**接口地址:** `GET /api/config`

**请求参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| dataId | string | 是 | 配置ID |
| group | string | 否 | 配置分组（默认DEFAULT_GROUP） |

**请求示例:**

```bash
curl "http://localhost:8080/api/config?dataId=application.yml&group=DEFAULT_GROUP"
```

**响应示例:**

```json
{
  "content": "server:\n  port: 8080\n"
}
```

#### 列出配置

**接口地址:** `GET /api/configs`

**请求参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| pageNo | int | 否 | 页码（默认1） |
| pageSize | int | 否 | 每页大小（默认10） |

**请求示例:**

```bash
curl "http://localhost:8080/api/configs?pageNo=1&pageSize=10"
```

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

#### 搜索配置

**接口地址:** `GET /api/search`

**请求参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| dataId | string | 否 | 配置ID（模糊匹配） |
| group | string | 否 | 配置分组（模糊匹配） |
| pageNo | int | 否 | 页码（默认1） |
| pageSize | int | 否 | 每页大小（默认10） |

**请求示例:**

```bash
curl "http://localhost:8080/api/search?dataId=application&pageNo=1&pageSize=10"
```

**响应示例:**

```json
{
  "totalCount": 5,
  "pageNumber": 1,
  "pagesAvailable": 1,
  "pageItems": [
    {
      "dataId": "application.yml",
      "group": "DEFAULT_GROUP",
      "content": "..."
    }
  ]
}
```

### 配置说明

**环境变量:**

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| NACOS_SERVER | Nacos服务地址 | `http://localhost:8848` |
| NACOS_NAMESPACE | 命名空间ID | `public` |
| NACOS_USERNAME | 用户名 | `nacos` |
| NACOS_PASSWORD | 密码 | `nacos` |

---

## 7. EIP-Plugs - 公网IP查询

### 功能概述

提供公网IP地址查询服务，通过多个公共API并发查询，确保结果准确性。

### 核心特性

- ✅ **多源查询** - 并发查询多个公共IP服务
- ✅ **快速响应** - 并发请求，返回最快结果
- ✅ **高可用性** - 多个数据源保证可用性
- ✅ **简单易用** - 单一接口获取公网IP

### 项目结构

```
eip-plugs/
├── main.go              # 主入口
├── common/
│   ├── logger.go       # 日志管理
│   └── response.go     # 统一响应
├── model/
│   └── ip.go           # IP模型
├── router/
│   └── router.go       # 路由管理
└── api/
    └── ip_handler.go   # IP查询接口
```

### 快速开始

#### 启动服务

```bash
go mod tidy
go run main.go
```

服务将在 `http://localhost:8070` 启动

### API接口

#### 获取公网IP

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

### 工作原理

服务会并发查询以下公共IP服务：

1. `https://ident.me`
2. `https://ipv4.icanhazip.com`
3. `http://myip.ipip.net/ip`

返回最快响应的结果，确保查询速度和准确性。

### 使用场景

- 服务器公网IP自动上报
- 网络环境检测
- IP地址变更监控
- CMDB资产信息自动更新

---

## 附录

### 通用配置说明

所有插件都支持以下通用配置方式：

1. **配置文件** - 适合开发环境
2. **环境变量** - 适合生产环境（推荐）
3. **命令行参数** - 适合临时调试

### 配置优先级

```
命令行参数 > 环境变量 > 配置文件 > 默认值
```

### Docker部署建议

#### 1. 创建Dockerfile

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

#### 2. 使用docker-compose

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
      - "8082:8082"
    environment:
      - DB_HOST=mysql
      - DB_PASSWORD=your-password
  
  redis-plugs:
    build: ./redis-plugs
    ports:
      - "8080:8080"
    environment:
      - REDIS_HOST=redis
      - REDIS_PORT=6379
```

### 监控和日志

所有插件都支持：

- ✅ 控制台日志输出
- ✅ 文件日志记录
- ✅ 日志级别控制（info/error）
- ✅ 日志自动清理

### 安全建议

1. **生产环境必须使用环境变量** - 不要将密码写入配置文件
2. **定期更新依赖** - 使用 `go mod tidy` 更新依赖
3. **限制网络访问** - 使用防火墙限制访问来源
4. **启用HTTPS** - 生产环境建议使用反向代理启用HTTPS
5. **日志脱敏** - 确保日志中不包含敏感信息

### 性能优化

1. **连接池配置** - 合理配置数据库/Redis连接池大小
2. **超时设置** - 设置合理的请求超时时间
3. **并发控制** - 使用goroutine控制并发数量
4. **缓存策略** - 对频繁查询的数据进行缓存

### 故障排查

#### 常见问题

1. **连接失败** - 检查网络连接和防火墙设置
2. **认证失败** - 检查用户名密码是否正确
3. **超时错误** - 增加超时时间或优化查询
4. **内存溢出** - 减少单次查询数据量

#### 日志查看

```bash
# 查看实时日志
tail -f logs/app.log

# 查看错误日志
grep ERROR logs/app.log

# 查看最近100行
tail -n 100 logs/app.log
```

### 联系方式

如有问题或建议，请联系开发团队。

---

**文档版本:** v1.0  
**最后更新:** 2025-01-19  
**维护者:** CMDB团队
