# SQL 测试脚本

这个目录包含了用于测试的 SQL 脚本。

## 文件说明

### init.sql - 初始化数据库

创建测试需要的表结构：
- `user_info` - 用户表
- `order_info` - 订单表
- `repayment_order` - 还款表
- `product_info` - 商品表
- `order_product` - 订单商品关联表

**使用方法**：

```bash
# 连接到 MySQL
mysql -h 192.168.6.2 -u root -p test

# 执行初始化脚本
source D:/project/go/sql-plugs/script/init.sql
```

或者：

```bash
mysql -h 192.168.6.2 -u root -p test < D:/project/go/sql-plugs/script/init.sql
```

### select.sql - 测试查询

包含 15 个不同复杂度的 SQL 查询，用于测试系统的查询能力：

1. ✅ 窗口函数查询
2. ✅ CTE 递归查询
3. ✅ 统计分析查询
4. ✅ 多表聚合查询
5. ✅ 子查询筛选
6. ✅ 嵌套多层子查询
7. ✅ CASE 逻辑计算
8. ✅ 联表+聚合+子查询
9. ✅ 联表+时间筛选
10. ✅ 嵌套联表查询
11. ✅ 分区聚合查询
12. ✅ 嵌套 CASE 计算
13. ✅ 相关子查询
14. ✅ 窗口排名查询
15. ✅ 全局分析查询

**使用方法**：

可以复制这些 SQL 到测试页面（http://localhost:8090）进行测试。

## 测试流程

### 步骤 1：初始化数据库

```bash
# 执行 init.sql 创建表结构
mysql -h 192.168.6.2 -u root -p test < script/init.sql
```

### 步骤 2：插入测试数据

需要手动插入一些测试数据，或者使用数据生成工具。

### 步骤 3：测试查询

1. 启动服务：`go run main.go`
2. 打开浏览器：http://localhost:8090
3. 从 `select.sql` 中选择查询语句
4. 粘贴到测试页面执行
5. 查看查询结果、数量和耗时

## 快速测试示例

在测试页面中测试以下查询：

**单个查询**：
```sql
SELECT * FROM user_info LIMIT 50
```

**批量查询**（用分号分隔）：
```sql
SELECT * FROM user_info;
SELECT * FROM order_info;
SELECT * FROM product_info
```

**复杂查询**：
```sql
SELECT u.username, COUNT(r.id) AS total_orders, 
       SUM(r.repay_amount) AS total_repaid
FROM user_info u 
JOIN repayment_order r ON u.id = r.user_id 
GROUP BY u.id 
HAVING total_orders > 3
```

## 注意事项

1. **数据准备**：`select.sql` 中的查询需要有相应的测试数据才能正常执行
2. **性能测试**：建议插入大量数据（如 10000+ 条）来测试性能
3. **LIMIT 限制**：所有查询自动限制返回最多 100 条记录
4. **COUNT 查询**：系统会自动执行 COUNT 查询获取真实总数
