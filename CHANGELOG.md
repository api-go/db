# 变更记录 - @go/db

## [1.0.9] - 2026-05-09
- **基础设施对齐**: 
    - 升级 `apigo.cc/go/log` 至 `v1.1.13`。
    - 为 `DBInfoLog` 和 `DBErrorLog` 实现 `Reset()` 方法，以遵循 `log` 的强制 Reset 契约。
    - 调整 `DBLog` 内的字段 `pos` 索引，从 `6` 开始紧凑排列，消除索引空洞。
- **测试增强**: 
    - 修复多个测试用例 (`TestSmartDelete`, `TestGenericQuery`, `TestTableProbing`, `TestVersionControl`) 中因使用 `sqlite://:memory:` DSN 导致的初始化失败问题。
    - 引入 `test_util.go` 和 `ResetAllForTest()`，确保测试间的全局状态隔离。

## [1.0.6] - 2026-05-05
### 优化
- **日志体系重构**: 
  - 引入 `DBInfoLog` 和 `DBErrorLog` 类型化日志，分别继承 `log.InfoLog` 和 `log.ErrorLog`。
  - 利用 `log` 包的新特性实现“零手动”调用栈捕获，业务端仅需关注错误信息和 SQL 现场。
  - 进一步解耦业务字段与日志元数据。
### 优化
- **日志自主化**: 
  - 将数据库日志逻辑从 `log` 包迁移至 `db` 包，实现日志格式与业务逻辑的深度绑定。
  - 自定义 `DBLog` 结构，利用 `log.GetEntry[DBLog]()` 对象池加速，并支持完整的错误堆栈捕获。
  - 接入 `log.Log` 的自动填充能力，精简元数据维护逻辑。

## [1.0.4] - 2026-05-04
### 优化
- **日志增强**：升级 `apigo.cc/go/log` 至 v1.0.1，并重构数据库日志逻辑，利用新版 `log.DB` API 直接支持错误字段和调用栈捕获，提升排障效率。

## [1.0.3] - 2026-05-04
### 新增
- **自动随机 ID (Auto Random ID)**：当表主键或唯一索引字段类型为 `char(8/10/12/14)` 且值为空时，自动填充分布式唯一 ID。
- **智能 ID 生成器**：自动适配数据库类型（MySQL 右旋散列、PostgreSQL 时间单调、SQLite 纯随机），优先使用 Redis 分布式生成器。

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
- **乐观锁与版本控制**：支持 `db.Update` 自动处理版本递增与冲突检测。
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
