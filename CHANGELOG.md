# 变更记录 - @go/db

## [1.0.2] - 2026-05-04
### 修复
- **PostgreSQL 增强**：补全了 `getTable` 中的元数据探测逻辑，使 `autoVersion` 和影子删除在 PostgreSQL 下可自动启用。
- **错误处理一致性**：统一了 `QueryResult` 与 `ExecResult` 的错误传播逻辑，确保 `r.Error` 在数据处理阶段也能正确记录。
- **单元测试修复**：修正了 `DB_test.go` 中因 SQLite 时区差异导致的 `TestInsertReplaceUpdateDelete` 偶发失败。

### 优化
- **性能提升**：在 `Base.go` 中引入了 `sync.Map` 缓存结构体反射解析结果，减少 SQL 生成过程中的反射开销。

## [1.0.1] - 2026-05-03
### 新增
- **架构 DSL (Schema-as-Code)**：支持通过文本 DSL 定义并自动同步数据库结构。
- **影子删除 (Shadow Deletion)**：支持 `SD` 标记，使用 `db.Remove` 自动将删除数据移动到 `_deleted` 后缀的备份表中。
- **乐观锁与版本控制**：支持 `ver` 标记，`db.Update` 自动处理版本递增与冲突检测。
- **泛型支持**：新增 `db.ToSlice[T]` 和 `db.To[T]`，提供类型安全的查询结果映射。
- **PostgreSQL 支持**：初步支持 PostgreSQL 的架构同步逻辑。
- **AI 友好文档**：新增 `db.SchemaMarkdown()` 自动生成 Markdown 格式的数据库模型文档。

### 优化
- 重构了 `makeResults` 逻辑，预计算 Struct 字段映射，显著提升大数据集下的查询性能。
- 完善了 SQLite 的 `DATETIME` 与 Go `time.Time` 的自动转换逻辑。
- 所有的文档和注释已本地化为中文。

### 修复
- 修复了 `Tx` 结构体中的拼写错误 `isCommitedOrRollbacked` 为 `isCommittedOrRollbacked`。
- 统一了全模块的参数命名规范：`requestSql` -> `query`，`wheres` -> `conditions`。
