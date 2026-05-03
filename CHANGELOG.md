# 变更记录 - @go/db

## [1.1.0] - 2026-05-03
### 新增
- **架构 DSL (Schema-as-Code)**：支持通过文本 DSL 定义并自动同步数据库结构。
- **影子删除 (Shadow Deletion)**：支持 `SD` 标记，使用 `db.Remove` 自动将删除数据移动到 `_deleted` 后缀的备份表中。
- **乐观锁与版本控制**：支持 `ver` 标记，`db.Update` 自动处理版本递增与冲突检测。
- **泛型支持**：新增 `db.ToSlice[T]` 和 `db.ToValue[T]`，提供类型安全的查询结果映射。
- **PostgreSQL 支持**：初步支持 PostgreSQL 的架构同步逻辑。
- **AI 友好文档**：新增 `db.SchemaMarkdown()` 自动生成 Markdown 格式的数据库模型文档。

### 优化
- 重构了 `makeResults` 逻辑，预计算 Struct 字段映射，显著提升大数据集下的查询性能。
- 完善了 SQLite 的 `DATETIME` 与 Go `time.Time` 的自动转换逻辑。
- 所有的文档和注释已本地化为中文。

### 修复
- 修复了 `Tx` 结构体中的拼写错误 `isCommitedOrRollbacked` 为 `isCommittedOrRollbacked`。
- 统一了全模块的参数命名规范：`requestSql` -> `query`，`wheres` -> `conditions`。
