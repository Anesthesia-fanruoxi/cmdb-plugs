# ES查询服务

## 项目结构

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

## 功能特性

1. **Kibana风格查询** - 支持Kibana语法，智能转换为ES查询
2. **时间排序查询** - 支持按时间升序/降序排序，默认为降序
3. **索引映射查询** - 获取索引列表和字段映射，自动限制最新10个
4. **滚动查询** - 大数据量遍历，支持init/continue/clear操作
5. **上下文查询** - 获取指定文档前后的相关记录
6. **环境变量支持** - 配置可通过环境变量覆盖，方便容器化部署
7. **控制台日志** - 日志输出到控制台，便于Docker日志收集
8. **统一响应格式** - 标准化的API响应
9. **标准HTTP请求** - 使用Go标准库，无第三方SDK依赖

## 快速开始

### 1. 配置ES连接

#### 方式一：配置文件（推荐开发环境）

编辑 `config/config.yml`:

```yaml
elasticsearch:
  host: "http://localhost:9200"
  username: "elastic"
  password: "your-password"
  timeout: 30

log:
  level: "info"  # 日志级别: info 或 error
```

#### 方式二：环境变量（推荐生产环境）

环境变量优先级高于配置文件，支持以下环境变量：

| 环境变量 | 说明 | 示例 |
|---------|------|------|
| `ES_HOST` | ES服务地址 | `http://localhost:9200` |
| `ES_USERNAME` | ES用户名 | `elastic` |
| `ES_PASSWORD` | ES密码 | `your-password` |
| `LOG_LEVEL` | 日志级别 | `info` 或 `error` |
| `LIMIT_MAX_SIZE` | ES返回最大条数限制 | `3000` (最大3000) |

**Linux/Mac 设置:**

```bash
export ES_HOST="http://localhost:9200"
export ES_USERNAME="elastic"
export ES_PASSWORD="your-password"
```

**Windows PowerShell 设置:**

```powershell
$env:ES_HOST="http://localhost:9200"
$env:ES_USERNAME="elastic"
$env:ES_PASSWORD="your-password"
```

**Docker 运行:**

```bash
docker run -d \
  -e ES_HOST="http://elasticsearch:9200" \
  -e ES_USERNAME="elastic" \
  -e ES_PASSWORD="your-password" \
  -p 8081:8081 \
  es-plugs:latest
```

### 2. 安装依赖

```bash
go mod tidy
```

### 3. 运行服务

```bash
go run main.go
```

服务将在 `http://localhost:8081` 启动

## 配置说明

### 配置优先级

配置加载顺序：**环境变量 > 配置文件 > 默认值**

- **配置文件是可选的**，如果不存在会使用默认值
- 环境变量优先级最高，会覆盖配置文件和默认值
- 未设置的环境变量不影响配置文件或默认值

**默认配置：**
```yaml
elasticsearch:
  host: "http://localhost:9200"
  username: "elastic"
  password: ""
  timeout: 30

log:
  level: "info"

limit:
  max_size: 3000  # ES返回最大条数限制
```

### 最佳实践

1. **开发环境**
   - 使用 `config.yml` 配置文件
   - 方便修改和版本控制（注意排除敏感信息）

2. **生产环境（推荐）**
   - **不使用配置文件**，仅通过环境变量配置
   - 提高安全性，密码不暴露在镜像中
   - 便于容器化部署（Docker/K8s）
   ```bash
   # 只需设置环境变量即可启动
   export ES_HOST="http://elasticsearch:9200"
   export ES_USERNAME="elastic"
   export ES_PASSWORD="your-password"
   ```

3. **混合使用**
   - 公共配置放在 `config.yml`
   - 敏感信息用环境变量覆盖
   ```yaml
   # config.yml 公共配置
   elasticsearch:
     host: "http://localhost:9200"
     timeout: 30
   
   # 环境变量设置敏感信息
   export ES_USERNAME="elastic"
   export ES_PASSWORD="your-password"
   ```

### 日志级别说明

支持两种日志级别：

- **info**（默认）- 输出所有日志，包括信息和错误
- **error** - 只输出错误日志，适合生产环境减少日志量

**使用场景：**
- 开发环境：使用 `info` 查看详细的请求和处理流程
- 生产环境：使用 `error` 只关注错误，减少日志输出

**设置方法：**

```bash
# 通过环境变量设置
export LOG_LEVEL=error

# 或在配置文件中设置
log:
  level: "error"
```

### 限制配置说明

**ES返回最大条数限制（LIMIT_MAX_SIZE）：**

- **默认值**：3000条
- **最大值**：3000条（硬限制，超过会被强制限制为3000）
- **说明**：即使请求size为5000，实际返回也不会超过3000条

**使用场景：**
- 防止单次查询返回过多数据导致内存溢出
- 通过环境变量灵活调整，无需重新编译

**设置方法：**

```bash
# 通过环境变量设置（会被限制在3000以内）
export LIMIT_MAX_SIZE=2000  # 实际生效为2000
export LIMIT_MAX_SIZE=5000  # 会被限制为3000

# 或在配置文件中设置
limit:
  max_size: 2000  # 设置为2000条
```

**示例：**
```bash
# 请求size=100，实际返回100条
# 请求size=3000，实际返回3000条
# 请求size=5000，实际返回3000条（被限制）
```

### 配置验证

启动时会在控制台输出配置信息（密码会脱敏），方便确认配置是否正确：

**使用配置文件时：**
```
2025-10-16 20:00:00 [INFO] 日志级别: info (info=所有日志, error=仅错误)
2025-10-16 20:00:00 [INFO] 配置加载完成 (配置文件可选，环境变量优先)
2025-10-16 20:00:00 [INFO] ES配置: host=http://localhost:9200, username=elastic, password=***, timeout=30
2025-10-16 20:00:00 [INFO] 服务启动在端口 :8081
```

**仅使用环境变量时（配置文件不存在）：**
```
2025-10-16 20:00:00 [INFO] 日志级别: info (info=所有日志, error=仅错误)
2025-10-16 20:00:00 [INFO] 配置加载完成 (配置文件可选，环境变量优先)
2025-10-16 20:00:00 [INFO] ES配置: host=http://elasticsearch:9200, username=elastic, password=***, timeout=30
2025-10-16 20:00:00 [INFO] 服务启动在端口 :8081
```

## API接口

### 查询ES（增强版）

**接口地址:** `POST /api/es/query`

**功能特性:**
- 支持完整ES查询DSL
- 支持分页（size、from）
- 支持排序（sort），默认按 `@timestamp` 倒序
- 支持聚合查询（aggregations）
- 支持字段过滤（_source）
- 支持高亮（highlight）

**基础查询示例:**

```json
{
  "index": "my-index",
  "query": {
    "match_all": {}
  }
}
```

**分页查询示例:**

```json
{
  "index": "my-index",
  "query": {
    "match": {
      "field_name": "search_value"
    }
  },
  "size": 20,
  "from": 0
}
```

**排序查询示例:**

```json
{
  "index": "my-index",
  "query": {
    "match_all": {}
  },
  "sort": [
    {
      "created_at": {
        "order": "desc"
      }
    }
  ]
}
```

**聚合查询示例:**

```json
{
  "index": "my-index",
  "query": {
    "match_all": {}
  },
  "size": 0,
  "aggregations": {
    "status_count": {
      "terms": {
        "field": "status.keyword"
      }
    }
  }
}
```

**字段过滤示例:**

```json
{
  "index": "my-index",
  "query": {
    "match_all": {}
  },
  "_source": ["field1", "field2"]
}
```

**高亮查询示例:**

```json
{
  "index": "my-index",
  "query": {
    "match": {
      "content": "关键词"
    }
  },
  "highlight": {
    "fields": {
      "content": {}
    }
  }
}
```

**响应示例:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "took": 5,
    "timed_out": false,
    "total": 100,
    "max_score": 1.0,
    "hits": [
      {
        "_index": "my-index",
        "_id": "1",
        "_score": 1.0,
        "_source": {
          "field1": "value1",
          "field2": "value2"
        }
      }
    ],
    "aggregations": {
      "status_count": {
        "buckets": [
          {
            "key": "active",
            "doc_count": 50
          }
        ]
      }
    }
  }
}
```

### 健康检查

**接口地址:** `GET /health`

**响应:** `OK`

### 获取字段能力信息

**接口地址:** `POST /api/es/field-caps`

**请求示例:**

```json
{
  "index_pattern": "my-index-*",
  "fields": "field1,field2"
}
```

或查询所有字段:

```json
{
  "index_pattern": "my-index-*",
  "fields": "*"
}
```

**响应示例:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "indices": ["my-index-1", "my-index-2"],
    "fields": {
      "field1": {
        "keyword": {
          "type": "keyword",
          "searchable": true,
          "aggregatable": true
        }
      }
    }
  }
}
```

### 获取索引列表

**接口地址:** `POST /api/es/indices`

**请求示例:**

```json
{
  "index_pattern": "my-index-*"
}
```

或获取所有索引:

```json
{
  "index_pattern": "*"
}
```

**响应示例:**

```json
{
  "code": 200,
  "message": "success",
  "data": [
    {
      "index": "my-index-1",
      "status": "open",
      "health": "green",
      "docs.count": "1000",
      "store.size": "1.2mb"
    },
    {
      "index": "my-index-2",
      "status": "open",
      "health": "green",
      "docs.count": "500",
      "store.size": "800kb"
    }
  ]
}
```

### 获取索引映射

**接口地址:** `POST /api/es/mappings`

**请求示例:**

```json
{
  "index_pattern": "my-index"
}
```

**响应示例:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "my-index": {
      "mappings": {
        "properties": {
          "field1": {
            "type": "keyword"
          },
          "field2": {
            "type": "text"
          }
        }
      }
    }
  }
}
```

### 滚动查询

**接口地址:** `POST /api/es/scroll`

**功能说明:** 用于大数据量遍历，支持三种操作：init（初始化）、continue（继续）、clear（清除）

**支持两种查询方式：**
1. **时间范围+关键词**（推荐）- 使用 start_time/end_time/keyword 参数，自动解析
2. **自定义查询** - 使用 query 参数，传递标准 ES DSL

**初始化滚动查询示例1（时间+关键词）:**

```json
{
  "action": "init",
  "index": "jxh_sms_*",
  "start_time": "2025-10-17 11:03:11",
  "end_time": "2025-10-17 11:33:11",
  "time_field": "sendTimeStamp",
  "keyword": "中国移动 or 中国联通",
  "size": 2000,
  "scroll_time": "1m",
  "_source": ["sendTimeStamp", "operator"]
}
```

**初始化滚动查询示例2（自定义查询）:**

```json
{
  "action": "init",
  "index": "my-index",
  "query": {
    "match_all": {}
  },
  "size": 1000,
  "scroll_time": "1m"
}
```

**响应示例:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "scroll_id": "DXF1ZXJ5QW5kRmV0Y2gBAAAAAAAAAD4WYm9laVYtZndUQlNsdDcwakFMNjU1QQ==",
    "query_time": 5,
    "total_hits": 10000,
    "actual_hits": 1000,
    "hits": [
      {
        "_index": "my-index",
        "_id": "1",
        "_score": 1.0,
        "_source": {
          "field1": "value1"
        }
      }
    ]
  }
}
```

**继续滚动查询示例:**

```json
{
  "action": "continue",
  "scroll_id": "DXF1ZXJ5QW5kRmV0Y2gBAAAAAAAAAD4WYm9laVYtZndUQlNsdDcwakFMNjU1QQ==",
  "scroll_time": "1m"
}
```

**清除滚动上下文示例:**

```json
{
  "action": "clear",
  "scroll_id": "DXF1ZXJ5QW5kRmV0Y2gBAAAAAAAAAD4WYm9laVYtZndUQlNsdDcwakFMNjU1QQ=="
}
```

### 上下文查询

**接口地址:** `POST /api/es/context`

**功能说明:** 获取指定文档ID前后的相关文档，常用于日志查看场景

**请求示例:**

```json
{
  "index": "my-index",
  "doc_id": "center-doc-id",
  "before": 10,
  "after": 10,
  "sort_field": "@timestamp",
  "_source": ["field1", "field2", "@timestamp"]
}
```

**响应示例:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "before": [
      {
        "_index": "my-index",
        "_id": "before-1",
        "_score": 0,
        "_source": {
          "field1": "value1",
          "@timestamp": "2024-01-01T10:00:00Z"
        }
      }
    ],
    "center": {
      "_index": "my-index",
      "_id": "center-doc-id",
      "_score": 0,
      "_source": {
        "field1": "center-value",
        "@timestamp": "2024-01-01T10:05:00Z"
      }
    },
    "after": [
      {
        "_index": "my-index",
        "_id": "after-1",
        "_score": 0,
        "_source": {
          "field1": "value3",
          "@timestamp": "2024-01-01T10:10:00Z"
        }
      }
    ],
    "total": 21,
    "before_total": 10,
    "after_total": 10,
    "took": 25
  }
}
```

## 日志管理

- 日志文件存储在 `./logs` 目录
- 文件命名格式: `app_2024-01-01.log`
- 自动清理超过1天的日志文件（可在配置文件中修改）
- 日志同时输出到控制台和文件

## 测试示例

使用curl测试:

```bash
# 查询数据
curl -X POST http://localhost:8081/api/es/query \
  -H "Content-Type: application/json" \
  -d '{
    "index": "my-index",
    "query": {
      "match_all": {}
    }
  }'

# 获取字段能力
curl -X POST http://localhost:8081/api/es/field-caps \
  -H "Content-Type: application/json" \
  -d '{
    "index_pattern": "my-index-*",
    "fields": "*"
  }'

# 获取索引列表
curl -X POST http://localhost:8081/api/es/indices \
  -H "Content-Type: application/json" \
  -d '{
    "index_pattern": "*"
  }'

# 获取索引映射
curl -X POST http://localhost:8081/api/es/mappings \
  -H "Content-Type: application/json" \
  -d '{
    "index_pattern": "my-index"
  }'
```

使用PowerShell测试:

```powershell
# 查询数据
$body = @{
    index = "my-index"
    query = @{
        match_all = @{}
    }
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8081/api/es/query" -Method Post -Body $body -ContentType "application/json"

# 获取字段能力
$body = @{
    index_pattern = "my-index-*"
    fields = "*"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8081/api/es/field-caps" -Method Post -Body $body -ContentType "application/json"

# 获取索引列表
$body = @{
    index_pattern = "*"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8081/api/es/indices" -Method Post -Body $body -ContentType "application/json"

# 获取索引映射
$body = @{
    index_pattern = "my-index"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8081/api/es/mappings" -Method Post -Body $body -ContentType "application/json"
```

## 常见问题

### 1. 环境变量不生效？

确保环境变量在启动服务前已设置：

```bash
# 查看环境变量
echo $ES_HOST          # Linux/Mac
echo $env:ES_HOST      # Windows PowerShell

# 设置后立即运行
export ES_HOST="http://localhost:9200" && go run main.go
```

### 2. 如何确认使用的是哪个配置？

查看启动日志，会输出实际使用的配置信息：

```
2025-10-16 20:00:00 [INFO] 配置加载成功
2025-10-16 20:00:00 [INFO] ES配置: host=http://localhost:9200, username=elastic, password=***, timeout=30
2025-10-16 20:00:00 [INFO] 日志配置: dir=./logs, max_age=1天
```

### 3. Docker 部署建议

创建 `.env` 文件：

```bash
ES_HOST=http://elasticsearch:9200
ES_USERNAME=elastic
ES_PASSWORD=your-password
```

使用 docker-compose：

```yaml
version: '3'
services:
  es-plugs:
    image: es-plugs:latest
    env_file: .env
    ports:
      - "8081:8081"
    volumes:
      - ./logs:/app/logs
```

### 4. Kubernetes 部署

使用 Secret 存储敏感信息：

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: es-plugs-secret
type: Opaque
stringData:
  ES_USERNAME: elastic
  ES_PASSWORD: your-password
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: es-plugs
spec:
  template:
    spec:
      containers:
      - name: es-plugs
        image: es-plugs:latest
        env:
        - name: ES_HOST
          value: "http://elasticsearch:9200"
        envFrom:
        - secretRef:
            name: es-plugs-secret
```

## 安全建议

1. ⚠️ **不要将密码提交到代码仓库**
   - 使用 `.gitignore` 排除 `config.yml`（如果包含密码）
   - 或在 `config.yml` 中留空，使用环境变量设置

2. 🔒 **生产环境必须使用环境变量**
   - 通过平台的密钥管理服务注入
   - 定期轮换密码

## 许可证

MIT License
