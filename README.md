# @go/db

> **维护者声明：** 本项目由 AI 维护。代码源自 `github.com/ssgo/db` 的重构，支持现代 Go 特性、内存安全防护、读写分离、全局版本同步及泛型优化。

## 🎯 设计哲学：约定优于配置

`@go/db` 遵循“约定优于配置”的设计哲学，旨在通过合理的默认行为和命名约定，简化数据库操作，同时保持强大的功能。

*   **隐式高级功能**：版本控制和软删除等高级功能是**自动启用**的，无需显式配置。
    - **版本控制**: 如果一个表包含名为 `autoVersion` 且类型为 `bigint unsigned` (`ubi`) 的字段，`Update` 和 `Insert` 操作将自动处理其版本递增和乐观锁。
    - **智能删除**: 如果存在一个名为 `[表名]_deleted` 的表，`Delete` 操作将自动执行**影子删除**（将数据移入该表）；否则，执行物理删除。
*   **全局版本号**：内置基于 Redis 的全局序列（自动降级为本地 Map），确保分布式环境下 `version` 绝对单调递增，为可靠的增量同步提供基础。
*   **架构即代码 (DSL)**：`SD` 标记现在仅用于**建表**时自动创建 `_deleted` 表，而运行时的删除行为由约定决定。

## 📦 安装

```bash
go get apigo.cc/go/db
```

## 🛠 API 指南

### 1. 核心方法
- **`GetDB(name string, logger *log.Logger) (*DB, error)`**
  - 获取数据库连接实例。`name` 对应 `db.json` 中的配置名。
- **`Sync(schema string) error`**
  - 解析 DSL 并同步数据库表结构。用于创建表（包括 `_deleted` 表）和索引。

### 2. 写操作 (返回 `(*ExecResult, error)`)
- **`Insert(table string, data any)`**: 插入数据。若表符合 `autoVersion` 约定，会自动注入新的全局版本号。
- **`Update(table string, data any, conditions string, args ...any)`**: 更新数据。若表符合 `autoVersion` 约定，自动递增版本号并应用乐观锁。
- **`Delete(table string, conditions string, args ...any)`**: **智能删除**。根据是否存在 `_deleted` 表自动选择物理删除或影子删除。

### 3. 读操作
- **`Query(query string, args ...any) (*QueryResult, error)`**: 执行查询。
- **`QueryResult` 结果处理**:
    - **泛型 API (推荐)**: `db.ToSlice[T](...)`, `db.ToValue[T](...)`
    - **链式方法**: `To(ptr)`, `MapResults()`, `ToKV(mapPtr)`, `IntOnR1C1()` 等。

### 4. 事务
```go
tx, err := db.Begin()
if err != nil { /* ... */ }
defer tx.CheckFinished()
// ... 事务操作 ...
tx.Commit()
```

## 📖 详细文档
- [架构 DSL 与版本同步指南](./DSL.md)
- [测试报告](./TEST.md)
- [版本变更记录](./CHANGELOG.md)
