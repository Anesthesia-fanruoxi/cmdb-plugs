# 🔍 ES Agent API 接口文档

> Elasticsearch 查询代理服务接口文档

---

## 📌 接口概览

| 接口名称 | 请求方式 | 接口地址 | 描述 |
|---------|---------|---------|------|
| 查询数据 | POST | `/api/elfk/search` | 支持 Kibana 风格语法的智能查询接口 |
| 索引映射 | GET | `/api/elfk/indices` | 获取索引列表及最新索引的字段映射 |
| 滚动查询 | POST | `/api/elfk/scroll` | 大数据量遍历，支持 init/continue/clear 操作 |
| 上下文查询 | POST | `/api/elfk/context` | 获取指定文档前后的相关记录 |
| 健康检查 | GET | `/health` | 服务健康状态检查 |

---

## 🔎 查询数据接口

### 基本信息

- **接口地址**: `/api/elfk/search`
- **请求方式**: `POST`
- **Content-Type**: `application/json`

### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|-------|------|------|------|------|
| index | string | ✅ | ES索引名，支持通配符 | `jxh_sms_*` |
| start_time | string | ✅ | 开始时间 | `2025-10-16 00:00:00` |
| end_time | string | ✅ | 结束时间 | `2025-10-17 00:00:00` |
| time_field | string | ✅ | 时间字段名（默认 `@timestamp`） | `timestamp` |
| keyword | string | ✅ | 关键词查询，支持复杂语法 | `level:error AND message:timeout` |
| size | int | ✅ | 返回记录数（默认50） | `100` |
| sort_order | string | ❌ | 时间排序方向（默认 `desc`） | `asc`（升序）或 `desc`（降序） |

### 请求示例

#### 1️⃣ 简单查询（查询所有）

```json
POST /api/elfk/search
{
  "index": "jxh_sms_sending_record_20251016",
  "start_time": "2025-10-16 00:00:00",
  "end_time": "2025-10-17 00:00:00",
  "time_field": "timestamp",
  "keyword": "*",
  "size": 50
}
```

#### 2️⃣ 关键词搜索

```json
{
  "index": "jxh_sms_*",
  "start_time": "2025-10-16 00:00:00",
  "end_time": "2025-10-17 00:00:00",
  "time_field": "timestamp",
  "keyword": "error",
  "size": 100
}
```

#### 3️⃣ 字段查询

```json
{
  "index": "jxh_sms_*",
  "start_time": "2025-10-16 00:00:00",
  "end_time": "2025-10-17 00:00:00",
  "time_field": "timestamp",
  "keyword": "status:failed",
  "size": 50
}
```

#### 4️⃣ 复杂逻辑查询

```json
{
  "index": "jxh_sms_*",
  "start_time": "2025-10-16 00:00:00",
  "end_time": "2025-10-17 00:00:00",
  "time_field": "timestamp",
  "keyword": "level:error AND message:\"timeout\" NOT user:admin",
  "size": 50
}
```

#### 5️⃣ 时间升序查询

```json
{
  "index": "jxh_sms_*",
  "start_time": "2025-10-16 00:00:00",
  "end_time": "2025-10-17 00:00:00",
  "time_field": "timestamp",
  "keyword": "*",
  "size": 50,
  "sort_order": "asc"
}
```

### 查询语法说明

#### 支持的操作符

| 操作符 | 说明 | 示例 |
|-------|------|------|
| `field:value` | 字段匹配 | `status:success` |
| `field=value` | 字段匹配（同上） | `status=success` |
| `field!=value` | 字段不匹配 | `status!=failed` |
| `"phrase"` | 精确短语匹配 | `"connection timeout"` |
| `AND` | 逻辑与 | `error AND database` |
| `OR` | 逻辑或 | `error OR warning` |
| `NOT` | 逻辑非 | `error NOT timeout` |
| `field:*` | 字段存在性 | `error_code:*` |
| `field!=*` | 字段不存在 | `error_code!=*` |

#### 查询语法示例

```
# 简单关键词搜索
error

# 字段查询
level:error
status=failed

# 精确匹配
message:"database connection timeout"

# 逻辑组合
level:error AND service:payment
level:error OR level:warning
error NOT timeout

# 字段存在性
error_code:*
response!=*

# 复杂组合
level:error AND (service:payment OR service:order) NOT user:system
```

### 响应格式

```json
{
  "code": 200,
  "message": "查询成功",
  "data": {
    "query_time": 3,
    "timed_out": false,
    "total_hits": 38011,
    "actual_hits": 1,
    "hits": [
      {
        "_index": "jxh_sms_sending_record_20251016",
        "_id": "abc123",
        "_score": 1.0,
        "_source": {
          "timestamp": 1729008000000,
          "phone": "13800138000",
          "status": "success",
          "message": "发送成功",
          "create_time": "2025-10-16 10:00:00"
        }
      }
    ]
  }
}
```

### 响应字段说明

| 字段名 | 类型 | 说明 |
|-------|------|------|
| code | int | 状态码（200成功） |
| message | string | 响应消息 |
| data.query_time | int | 查询耗时（毫秒） |
| data.timed_out | bool | 是否超时 |
| data.total_hits | int | 匹配记录总数（真实总数，不限制10000） |
| data.actual_hits | int | 实际返回的记录条数 |
| data.hits | array | 命中记录数组 |
| data.hits[]._index | string | 索引名 |
| data.hits[]._id | string | 文档ID |
| data.hits[]._score | float | 相关性评分 |
| data.hits[]._source | object | 文档内容 |

---

## 📋 索引映射接口

### 基本信息

- **接口地址**: `/api/elfk/indices`
- **请求方式**: `GET`

### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|-------|------|------|------|------|
| index | string | ✅ | 索引模式，支持通配符 | `jxh_sms_sending_record_*` |

### 请求示例

```
GET /api/elfk/indices?index=jxh_sms_sending_record_*
```

### 响应格式

```json
{
  "code": 200,
  "message": "获取索引映射成功",
  "data": {
    "indices": [
      "jxh_sms_sending_record_20251014",
      "jxh_sms_sending_record_20251015",
      "jxh_sms_sending_record_20251016"
    ],
    "fields": {
      "properties": {
        "timestamp": {
          "type": "long"
        },
        "phone": {
          "type": "text"
        },
        "status": {
          "type": "keyword"
        },
        "message": {
          "type": "text"
        },
        "create_time": {
          "type": "date"
        }
      }
    }
  }
}
```

### 响应字段说明

| 字段名 | 类型 | 说明 |
|-------|------|------|
| code | int | 状态码（200成功） |
| message | string | 响应消息 |
| data.indices | array | 匹配的所有索引列表（按名称排序） |
| data.fields | object | 最新索引的字段映射信息 |
| data.fields.properties | object | 字段列表 |
| data.fields.properties.{field}.type | string | 字段类型（text/keyword/long/date等） |

### 功能说明

1. **自动排序** - 索引列表按名称排序，确保获取最新索引
2. **简化映射** - 只返回字段名和类型，去除冗余信息
3. **一次请求** - 同时返回索引列表和字段映射，减少请求次数
> 实现逻辑调用了es 的2个接口来组合实现的，一个indeices一个mapping
---

## 🔄 滚动查询接口

### 基本信息

- **接口地址**: `/api/elfk/scroll`
- **请求方式**: `POST`
- **Content-Type**: `application/json`

### 功能说明

滚动查询用于遍历大量数据，支持三种操作：
- **init** - 初始化滚动查询，返回第一批数据和 scroll_id
- **continue** - 使用 scroll_id 继续获取下一批数据
- **clear** - 清除滚动上下文，释放资源

### 操作1️⃣: 初始化滚动查询 (init)

#### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|-------|------|------|------|------|
| action | string | ✅ | 操作类型 | `init` |
| index | string | ✅ | 索引名（支持通配符） | `jxh_sms_*` |
| start_time | string | ❌ | 开始时间 | `2025-10-17 11:03:11` |
| end_time | string | ❌ | 结束时间 | `2025-10-17 11:33:11` |
| time_field | string | ❌ | 时间字段名（默认@timestamp） | `sendTimeStamp` |
| keyword | string | ❌ | 关键词搜索（支持AND/OR/NOT） | `中国移动 or 中国联通` |
| query | object | ❌ | 自定义ES查询（优先级低于时间+关键词） | `{"match_all": {}}` |
| size | int | ❌ | 每批返回记录数（默认100） | `2000` |
| scroll_time | string | ❌ | 上下文保持时间（默认1m） | `5m` |
| sort | array | ❌ | 排序规则 | `[{"timestamp": "desc"}]` |
| _source | array/object | ❌ | 字段过滤 | `["sendTimeStamp", "operator"]` |

#### 请求示例1：使用时间范围+关键词（推荐）

```json
POST /api/elfk/scroll
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

#### 请求示例2：使用自定义查询

```json
POST /api/elfk/scroll
{
  "action": "init",
  "index": "jxh_sms_sending_record_*",
  "query": {
    "match_all": {}
  },
  "size": 1000,
  "scroll_time": "5m"
}
```

#### 参数优先级说明

**查询构建优先级：**
1. **时间范围+关键词**（`start_time` / `end_time` / `keyword`）- 优先级最高
   - 使用 QueryBuilder 智能解析，支持复杂语法
   - 适用场景：大多数业务查询
2. **自定义查询**（`query`）- 优先级次之
   - 直接传递给 ES 的查询 DSL
   - 适用场景：需要精确控制查询的高级用户
3. **默认查询**（`match_all`）- 兜底方案
   - 如果以上都未提供，返回所有数据

**使用建议：**
- 💡 推荐使用示例1的方式（时间+关键词），更简洁易用
- 🔧 需要复杂聚合、嵌套查询时使用示例2（自定义查询）
- ⚠️ 不要同时提供多种查询方式，会按优先级只使用一种

#### 响应示例

```json
{
  "code": 200,
  "message": "滚动查询成功",
  "data": {
    "scroll_id": "DXF1ZXJ5QW5kRmV0Y2gBAAAAAAAAAD4WYm9laVYtZndUQlNsdDcwakFMNjU1QQ==",
    "query_time": 15,
    "total_hits": 50000,
    "actual_hits": 1000,
    "hits": [
      {
        "_index": "jxh_sms_sending_record_20251016",
        "_id": "1",
        "_score": 1.0,
        "_source": {
          "timestamp": 1729008000000,
          "phone": "13800138000"
        }
      }
    ]
  }
}
```

### 操作2️⃣: 继续滚动查询 (continue)

#### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|-------|------|------|------|------|
| action | string | ✅ | 操作类型 | `continue` |
| scroll_id | string | ✅ | 滚动ID | `DXF1ZXJ5QW5k...` |
| scroll_time | string | ❌ | 上下文保持时间（默认1m） | `5m` |

#### 请求示例

```json
POST /api/elfk/scroll
{
  "action": "continue",
  "scroll_id": "DXF1ZXJ5QW5kRmV0Y2gBAAAAAAAAAD4WYm9laVYtZndUQlNsdDcwakFMNjU1QQ==",
  "scroll_time": "5m"
}
```

#### 响应示例

```json
{
  "code": 200,
  "message": "滚动查询成功",
  "data": {
    "scroll_id": "DXF1ZXJ5QW5kRmV0Y2gBAAAAAAAAAD4WYm9laVYtZndUQlNsdDcwakFMNjU1QQ==",
    "query_time": 3,
    "total_hits": 50000,
    "actual_hits": 1000,
    "hits": [
      {
        "_index": "jxh_sms_sending_record_20251016",
        "_id": "1001",
        "_score": 1.0,
        "_source": {
          "timestamp": 1729008100000,
          "phone": "13800138001"
        }
      }
    ]
  }
}
```

**注意**: 当 `hits` 为空数组时，`actual_hits` 为 0，表示已遍历完所有数据，`scroll_id` 会被自动清空。

### 操作3️⃣: 清除滚动上下文 (clear)

#### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|-------|------|------|------|------|
| action | string | ✅ | 操作类型 | `clear` |
| scroll_id | string | ✅ | 滚动ID | `DXF1ZXJ5QW5k...` |

#### 请求示例

```json
POST /api/elfk/scroll
{
  "action": "clear",
  "scroll_id": "DXF1ZXJ5QW5kRmV0Y2gBAAAAAAAAAD4WYm9laVYtZndUQlNsdDcwakFMNjU1QQ=="
}
```

#### 响应示例

```json
{
  "code": 200,
  "message": "滚动查询成功",
  "data": {
    "scroll_id": "DXF1ZXJ5QW5kRmV0Y2gBAAAAAAAAAD4WYm9laVYtZndUQlNsdDcwakFMNjU1QQ==",
    "cleared": true
  }
}
```

### 使用流程

```
1. Init (初始化)
   ↓ 返回 scroll_id 和第一批数据
   
2. Continue (继续获取)
   ↓ 使用 scroll_id 获取下一批
   ↓ 重复此步骤直到 hits 为空
   
3. Clear (清除)或自动清除
   ↓ 释放服务器资源
```

---
## 📍 上下文查询接口

### 基本信息

- **接口地址**: `/api/elfk/context`
- **请求方式**: `POST`
- **Content-Type**: `application/json`

### 功能说明

上下文查询用于获取指定文档前后的相关记录，常用于日志查看场景，快速了解某条日志前后发生了什么。

### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|-------|------|------|------|------|
| index | string | ✅ | 索引名 | `axh-axh-app-info-2025.10.16` |
| doc_id | string | ✅ | 中心文档ID | `8M0U7ZkB9a7kPIMCgj-u` |
| before | int | ✅ | 获取前面多少条（默认0） | `20` |
| after | int | ✅ | 获取后面多少条（默认0） | `20` |
| sort_field | string | ✅ | 排序字段（默认@timestamp） | `timestamp` |
| _source | array/object | ✅ | 字段过滤 | `["field1", "field2"]` |

### 请求示例

```json
POST /api/elfk/context
{
  "index": "axh-axh-app-info-2025.10.16",
  "doc_id": "8M0U7ZkB9a7kPIMCgj-u",
  "before": 20,
  "after": 20,
  "sort_field": "timestamp"
}
```

### 响应示例

```json
{
  "code": 200,
  "message": "上下文查询成功",
  "data": {
    "before": [
      {
        "_index": "axh-axh-app-info-2025.10.16",
        "_id": "before-1",
        "_score": 0,
        "_source": {
          "timestamp": 1729008000000,
          "message": "用户登录",
          "level": "info"
        }
      },
      {
        "_index": "axh-axh-app-info-2025.10.16",
        "_id": "before-2",
        "_score": 0,
        "_source": {
          "timestamp": 1729008050000,
          "message": "开始处理请求",
          "level": "info"
        }
      }
    ],
    "center": {
      "_index": "axh-axh-app-info-2025.10.16",
      "_id": "8M0U7ZkB9a7kPIMCgj-u",
      "_source": {
        "timestamp": 1729008100000,
        "message": "处理超时",
        "level": "error"
      }
    },
    "after": [
      {
        "_index": "axh-axh-app-info-2025.10.16",
        "_id": "after-1",
        "_score": 0,
        "_source": {
          "timestamp": 1729008150000,
          "message": "重试处理",
          "level": "warn"
        }
      },
      {
        "_index": "axh-axh-app-info-2025.10.16",
        "_id": "after-2",
        "_score": 0,
        "_source": {
          "timestamp": 1729008200000,
          "message": "处理完成",
          "level": "info"
        }
      }
    ],
    "total": 45,
    "before_total": 20,
    "after_total": 20,
    "took": 25
  }
}
```

### 响应字段说明

| 字段名 | 类型 | 说明 |
|-------|------|------|
| before | array | 中心文档之前的记录（按时间升序） |
| center | object | 中心文档 |
| after | array | 中心文档之后的记录（按时间升序） |
| total | int | 总记录数（before + 1 + after） |
| before_total | int | 前面的记录数 |
| after_total | int | 后面的记录数 |
| took | int | 查询耗时（毫秒） |

### 工作原理

```
时间线: ←─────────── [before] ──── [center] ──── [after] ────────────→

1. 获取中心文档（通过 doc_id）
2. 提取排序字段的值（如 timestamp: 1729008100000）
3. 查询 before 条记录（timestamp < 1729008100000，降序后反转）
4. 查询 after 条记录（timestamp > 1729008100000，升序）
5. 返回组合结果
```
