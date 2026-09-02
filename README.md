# @go/db

> **维护者声明：** 本项目由 AI 维护。代码源自 `github.com/ssgo/db` 的重构，支持现代 Go 特性、内存安全防护、读写分离、全局版本同步及泛型优化。

## 🎯 设计哲学：约定优于配置

`@go/db` 遵循“约定优于配置”的设计哲学，旨在通过合理的默认行为和命名约定，简化数据库操作，同时保持强大的功能。

*   **隐式高级功能**：版本控制和软删除等高级功能是**自动启用**的，无需显式配置。
    - **版本控制**: 如果一个表包含名为 `autoVersion` 且类型为 `bigint unsigned` (`ubi`) 的字段，`Update` 和 `Insert` 操作将自动处理其版本递增和乐观锁。
    - **自动随机 ID**: 当字段类型为 `char(8/10/12/14)` 且为 `PRIMARY KEY` 或 `UNIQUE` 时，`Insert/Replace` 操作若发现该字段为空，将自动填充唯一 ID（优先使用 Redis 分布式 ID）。
    - **智能删除**: 如果存在一个名为 `[表名]_deleted` 的表，`Delete` 操作将自动执行**影子删除**（将数据移入该表）；否则，执行物理删除。
*   **全局版本号**：内置基于 Redis 的全局序列（自动降级为本地 Map），确保分布式环境下 `version` 绝对单调递增，为可靠的增量同步提供基础。
*   **架构即代码 (DSL)**：`SD` 标记现在仅用于**建表**时自动创建 `_deleted` 表，而运行时的删除行为由约定决定。
    - 索引标记支持 `I<n>`、`U<n>` 与 `TI<n>` 分组；相同分组号的字段组成普通、唯一或全文联合索引。

## 📦 安装

```bash
go get apigo.cc/go/db
```

## 🛠 API 指南

### 1. 核心方法
- **`SetConfig(name, setting string)`**
  - 动态设置数据库配置（不依赖配置文件），可通过别名获取连接。
- **`GetDB(name string, logger *log.Logger) *DB`**
  - 获取数据库连接实例。`name` 可以是别名、`db.json` 中的配置名，也可以是标准 DSN（如 `mysql://user:pwd@host:port/db` 或 `sqlite://test.db`）。
- **`Sync(schema string) error`**
  - 解析 DSL 并同步数据库表结构。用于创建表（包括 `_deleted` 表）和索引。详见 [架构 DSL 指南](./DSL.md)。

### 2. 写操作 (返回 `*ExecResult`)
- **`Insert/Replace(table string, data any)`**: 插入或替换数据。若表包含 `autoVersion` 字段，会自动注入初始版本号。
- **`Update(table string, data any, conditions string, args ...any)`**: 更新数据。若表包含 `autoVersion` 字段，自动递增版本号并应用乐观锁。
- **`Delete(table string, conditions string, args ...any)`**: **智能删除**。根据是否存在 `_deleted` 表自动选择物理删除或影子删除。

#### 结果判定 (`ExecResult`)
```go
res := dbInst.Insert("users", newUser)
if res.Error != nil { /* 发生 SQL 错误 */ }
count := res.Changes() // 受影响行数
id := res.Id()         // 获取自增 ID
```

### 3. 读操作 (返回 `*QueryResult`)
- **`Query(query string, args ...any)`**: 执行查询。
- **结果处理 (QueryResult)**:
    - **泛型绑定 (推荐)**: `db.To[T](res)`, `db.ToSlice[T](res)`
    - **KV 映射**: `res.ToKV(&mapObj)` 将前两列自动转为 Map。
    - **快捷取值**: `IntOnR1C1()`, `StringOnR1C1()`, `MapOnR1()`, `StringsOnC1()` 等。
    - **错误感知**: 所有结果方法都会同步更新 `res.Error`，可链式调用后统一判断。

## 🔐 安全与加密

我们极致注重数据安全：
- **密码防御**: 内存中的数据库密码受 `safe.SafeBuf` 保护，防止通过内存 Dump 获取明文。
- **配置加密**: 建议在 `db.json` 中使用密文存储敏感信息。
- **TODO: sskey 集成**: 计划引入 `sskey` 工具，实现生产环境密钥的统一托管与自动解密。

## 🏗 架构即代码 (DSL 示例)

我们鼓励通过 DSL 定义表结构，实现“修改代码即修改表”。

```go
schema := `
== Default ==
users SD // 用户表，开启影子删除
  id AI // 自增 ID
  name v50 U // 字符串(50)，唯一索引
  autoVersion ubi // 自动版本号
  status ti // 状态 (TinyInt)
`
dbInst.Sync(schema) // 自动创建 users 和 users_deleted 表及索引
```


### 4. 事务
```go
tx := dbInst.Begin()
if tx.Error != nil { /* 处理错误 */ }
defer tx.CheckFinished() // 自动处理未提交的 Rollback

tx.Insert("users", newUser)
if tx.Error == nil {
    tx.Commit()
}
```


## 📖 详细文档
- [架构 DSL 与版本同步指南](./DSL.md)
- [测试报告](./TEST.md)
- [版本变更记录](./CHANGELOG.md)
