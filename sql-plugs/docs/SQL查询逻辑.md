# SQL 查询逻辑文档

## 概述

本系统提供安全、智能的SQL查询功能，包含完整的SQL解析、风险评估、执行控制流程。

**核心原则：所有查询结果最多返回100条数据，但会返回真实总数（total）。**

## 查询流程

```
用户输入SQL
    ↓
1. 获取SQL
    ↓
2. SQL格式化（规范化空白字符）
    ↓
3. 获取SQL类型，确保是DQL
    ↓
4. 判断风险程度
    ↓
5. 根据风险等级处理
    ├─ 高风险 ──→ 执行COUNT ──→ 添加LIMIT 100 ──→ 执行查询
    │                                              ↓
    │                                         返回100条 + COUNT总数
    │
    └─ 低/中风险 ──→ 直接执行原SQL ──→ 遍历全部结果
                                        ↓
                                   返回100条 + 遍历总数
```

## 详细步骤

### 1. 获取SQL

- 支持多条SQL语句（分号分隔）
- 自动过滤注释（`--`、`#`、`/* */`）
- 字符串内的注释符号不会被误处理

**调用方法：** `common.SplitSQLStatements(sql)`

### 2. SQL格式化

将所有空白字符（换行、制表符等）规范化为单个空格，确保SQL类型识别准确。

**调用方法：** `common.NormalizeWhitespace(sql)`

**示例：**
```sql
-- 输入
UPDATE
  table
SET col = 1

-- 输出
UPDATE table SET col = 1
```

### 3. 获取SQL类型

识别SQL语句类型并分类：

| 类型 | 分类 | 说明 |
|------|------|------|
| SELECT, WITH | DQL | 数据查询 |
| INSERT, UPDATE, DELETE | DML | 数据操作 |
| CREATE, ALTER, DROP | DDL | 数据定义 |
| SHOW, DESCRIBE, EXPLAIN | OTHER | 其他查询 |

**调用方法：**
- `common.GetSQLType(sql)` - 获取类型
- `common.GetSQLCategory(sqlType)` - 获取分类

**查询接口限制：** 只允许 DQL 和 OTHER 类型

### 4. 判断风险程度

根据SQL特性分析风险等级：

| 风险等级 | 条件 | 处理策略 |
|----------|------|----------|
| **低风险** | 有LIMIT、WHERE、聚合函数(无GROUP BY)、HAVING | 不执行COUNT |
| **中风险** | 有JOIN、GROUP BY、DISTINCT | 查询全部，限制返回量 |
| **高风险** | 无任何过滤条件 | 强制LIMIT，执行COUNT |

**调用方法：**
- `common.AnalyzeSQLFeatures(sql)` - 分析SQL特性
- `common.AssessQueryRisk(sql, features)` - 评估风险

**SQL特性检测：**
```go
type SQLFeatures struct {
    HasWhere     bool   // WHERE条件
    HasJoin      bool   // JOIN关联
    HasGroupBy   bool   // GROUP BY
    HasHaving    bool   // HAVING
    HasOrderBy   bool   // ORDER BY
    HasDistinct  bool   // DISTINCT
    HasSubquery  bool   // 子查询
    HasUnion     bool   // UNION
    HasAggregate bool   // 聚合函数
    HasCTE       bool   // CTE(WITH)
    JoinType     string // JOIN类型
    JoinCount    int    // JOIN数量
}
```

### 5. 风险处理策略

**所有查询最多返回100条数据，但total字段返回真实总数。**

#### 低风险（low）和中风险（medium）
```
条件：有WHERE/JOIN/GROUP BY/DISTINCT/聚合函数等

处理方式：
1. 直接执行原始SQL（不添加LIMIT）
2. MySQL返回全部数据
3. Go程序遍历全部结果，统计真实总数
4. 只保存前100条数据返回给客户端

返回：rows(最多100条) + total(遍历得到的真实总数)
```

#### 高风险（high）
```
条件：SELECT * FROM table（无任何过滤条件）

处理方式：
1. 执行 COUNT(*) 查询获取真实总数（带10秒超时）
2. 原始SQL自动添加 LIMIT 100
3. 执行添加LIMIT后的SQL
4. 返回查询结果

返回：rows(最多100条) + total(COUNT得到的真实总数)
```

**调用方法：**
- `common.ProcessSQLLimit(sql)` - 处理LIMIT限制
- `common.BuildCountSQL(sql)` - 构建COUNT查询

### 6. 执行查询

- 支持查询取消（KILL QUERY）
- 带超时的COUNT查询（10秒）
- 自动设置字符集（utf8mb4）

## 核心方法一览

### common 包

| 文件 | 方法 | 功能 |
|------|------|------|
| `sqlComment.go` | `SplitSQLStatements` | 按分号分割SQL |
| | `RemoveSQLComments` | 移除所有注释 |
| | `AssessQueryRisk` | 评估查询风险 |
| `sqlutils.go` | `NormalizeWhitespace` | 规范化空白字符 |
| | `ProcessSQLLimit` | 处理LIMIT限制 |
| | `GetUserOriginalLimit` | 获取用户LIMIT值 |
| | `BuildCountSQL` | 构建COUNT查询 |
| | `HasFilterConditions` | 检测过滤条件 |
| | `IsReadOnlySQL` | 检查是否只读 |
| `sqlAnalyzeFeatures.go` | `GetSQLType` | 获取SQL类型 |
| | `GetSQLCategory` | 获取SQL分类 |
| | `AnalyzeSQLFeatures` | 分析SQL特性 |

### api 包

| 文件 | 方法 | 功能 |
|------|------|------|
| `search.go` | `SQLSearchHandler` | 查询接口处理器 |
| | `executeQueryWithRisk` | 根据风险执行查询 |
| `searchQuery.go` | `executeSingleQueryWithContext` | 执行单条查询 |
| | `executeCountWithTimeout` | 带超时COUNT |
| `analyze.go` | `SQLAnalyzeHandler` | 分析接口处理器 |

## 返回结果说明

```json
{
  "rows": [...],    // 查询结果，最多100条
  "total": 12345,   // 真实总数
  "columns": [...], // 列名
  "took": 50        // 耗时(ms)
}
```

- `rows`: 实际返回的数据行，最多100条
- `total`: 符合条件的真实总数（可能远大于100）

## 配置常量

```go
const (
    DefaultLimit = 100  // 默认返回记录数
    MaxLimit     = 1000 // 最大返回记录数
)
```

## 示例

### 低风险查询
```sql
SELECT * FROM users WHERE id = 1
-- 风险: low (有WHERE条件)
-- 处理: 直接执行，遍历全部结果获取总数，返回最多100条
```

### 中风险查询
```sql
SELECT * FROM users u JOIN orders o ON u.id = o.user_id
-- 风险: medium (有JOIN)
-- 处理: 直接执行，遍历全部结果获取总数，返回最多100条
```

### 高风险查询
```sql
SELECT * FROM users
-- 风险: high (无过滤条件)
-- 处理: 执行COUNT获取总数，自动添加LIMIT 100后执行查询
```

### 聚合查询
```sql
SELECT COUNT(*) FROM users
-- 风险: low (聚合函数，结果只有1行)
-- 处理: 直接执行，结果本身就是1行
```
