# 变更记录 - @go/db

## [1.0.12] - 2026-05-17
- **SQLite 极致优化 (超高并发支持)**:
    - **读写分离与零锁读取**: 读操作 (`Query`) 实现零锁定，配合 WAL 模式彻底解决读写互斥问题；写操作由应用层 `sync.Mutex` 统一排队，规避 `database is locked` 错误。
    - **临界区最小化**: 将 FTS 重写、参数 JSON 化 (`flatArgs`) 及日志记录移出锁保护区，极大缩短了写锁持有时间。
    - **工业级默认配置**: 
        - 自动启用 `WAL` 模式、`NORMAL` 同步、`MEMORY` 临时存储及 `busy_timeout(5000)`。
        - 引入 **动态 Mmap**：根据系统内存自动设置 `mmap_size` (最大 30GB 或物理内存的 1/4)，使大数据量访问接近内存速度。
        - 默认 `MaxOpenConns` 提升至 100，优化多线程只读性能。
- **稳定性**:
    - 为 `Stmt` (预处理语句) 增加写锁保护。
    - 优化事务锁机制，支持事务内的锁自动追踪与释放。

## [1.0.11] - 2026-05-13
- **基础设施对齐**:
    - 移除 `encoding/json` 原生依赖，全面切换至 `apigo.cc/go/cast.UnmarshalJSON` 以增强类型兼容性。
    - 移除测试代码中对 `os.Remove` 的直接调用，统一切换至 `apigo.cc/go/file.Remove` 以支持隔离文件系统。
- **稳定性增强**:
    - 验证并更新了测试用例，确保在 Go 1.25.0 环境下的执行稳定性。
    - 更新 `TEST.md` 性能基准，反映基础设施对齐后的最新指标。

## [1.0.10] - 2026-05-10
- **功能增强**:
    - 全面支持“复杂标识符”：改进了 `LIKE` 拦截逻辑中的正则表达式，支持带引号（`` ` ``, `"`, `'`, `[]`）和特殊字符（如 `-`）的表名与字段名。
    - 优化 `cleanIdentifier`：能够更精准地剥离多段式标识符（如 `table.column`）中的包装引号。
    - 增强 `getFTSMatchSQLParts` 和 `extractTableName`：确保在各种引用风格下均能正确定位影子列和源表。
- **基础设施对齐**:
    - 升级 `apigo.cc/go/log` 至 `v1.3.2`。
- **测试增强**:
    - 新增 `TestComplexIdentifierFTS` 验证复杂标识符下的全文检索重定向。
    - 修复并增强 `TestAutonomousFTS` 以支持多种引用风格的兼容性测试。

## [1.3.0] - 2026-05-12
- **基础设施对齐**: 
    - 官方发布 v1.3.0 对齐版本。

## [1.0.11] - 2026-05-11
- **基础设施对齐**: 
    - 最终基础设施对齐。

## [1.0.10] - 2026-05-10
- **基础设施对齐**: 
    - 升级 `apigo.cc/go/redis` 至 `v1.0.8`。
    - 验证了 `redis.NewIDMaker` API 变更后的集成稳定性。
    - 增强了 ID 生成器的集成测试，覆盖了 Redis 路径。

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
