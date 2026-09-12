# 变更记录 - @go/db

## [v1.5.18] - 2026-09-12
- 新增 PostgreSQL/pgx 应用层驱动支持和跨方言 SQL、Schema、索引兼容。
- `mysql`、`postgres` 及 PostgreSQL 别名支持零配置简写；host 和端口分别默认到本机回环地址及协议标准端口。
- MySQL/PostgreSQL 目标库不存在且账户有权限时自动创建并重连。
- MySQL 零配置在 Homebrew MariaDB 的 TCP root 认证失败时，可受控回退到本机 Unix socket 系统用户；显式连接不回退。
- MySQL Unix socket 参数、长字符串索引前缀和 PostgreSQL 大小写标识符、占位符、字段类型均增加兼容。
- 依赖升级到 `config v1.5.8`、`redis v1.5.18`。

## [v1.5.17] - 2026-09-12
- 修复以 `::` 开头的 IPv6 普通值在写入时被误判为旧式 `:SQL expression` 的问题。

## [v1.5.15] - 2026-09-05
- **依赖对齐**: 补齐 `config`、`file`、`log` 与 `redis` 的级联版本。

## [v1.5.14] - 2026-09-05
- **依赖对齐**: 升级全部 `apigo.cc/go/*` 依赖至当前已发布版本。

## [v1.5.13] - 2026-09-04
- 新增 SQLite Tx.Sync：复用事务连接执行 Schema DDL，提交后才使共享 Schema 缓存失效；传播提交和全文结构创建错误。
- 新增 TableStruct.Indexes 及独立索引声明 `a,b U`、`a,-b I`、`a I`，名称自动生成；SQLite 同步识别各列方向。
- 字段类型必填，精确校验简写与可枚举的原生类型；索引引用在执行全部表的 DDL 前校验；删除 `I<n>`/`U<n>`/`TI<n>` 数字分组语法。

## [1.5.12] - 2026-09-03
- **全文索引分组**: Schema DSL 支持通过 `TI<n>` 为全文索引指定分组号，与普通索引和唯一索引的分组语法保持一致。
- **依赖对齐**: 内部 `apigo.cc/go/*` 依赖全部升级至公共索引中的最新版本，并使用 `redis v1.5.14`。

## [1.5.11] - 2026-08-22
- SQLite schema 同步支持删除定义中已经移除的字段，并同步处理影子删除表。

## [1.5.10] - 2026-08-13
- 修复 schema 同步中的字段转义解析。

## [1.5.9] - 2026-07-18
- **依赖校验修复**: 升级 `redis` 至 `v1.5.11`，并将 `redigo` 固定为稳定版 `v1.9.3`，移除撤回版本。
- **依赖对齐**: 升级 `cast` 至 `v1.5.4`、`crypto` 至 `v1.5.4`、`encoding` 至 `v1.5.5`、`id` 至 `v1.5.6`。

## [1.5.7] - 2026-06-21
- **错误堆栈重构**:
  - 重构 `js_export.go`，将匿名函数改写为包级具名函数。
  - 在返回错误或向结果结构注入错误时，使用 `jsmod.MakeError` 动态包裹，以携带完整的调用堆栈。
- **依赖对齐**:
  - 升级依赖 `jsmod` 至 `v1.5.3`, `cast` 至 `v1.5.3`, `rand` 至 `v1.5.3`, `encoding` 至 `v1.5.4`, `shell` 至 `v1.5.3`, `safe` 至 `v1.5.2`, `id` 至 `v1.5.4`, `crypto` 至 `v1.5.3`, `file` 至 `v1.5.5`, `config` 至 `v1.5.3`, `log` 至 `v1.5.8`, `redis` 至 `v1.5.6`。

## [1.5.6] - 2026-06-11
- **依赖对齐**:
  - 对齐基础设施依赖到 v1.5.x。

## [1.5.5] - 2026-06-10
- **优化**:
  - 导出 PascalCase 格式的 JS 接口并平铺默认实例。

## [1.5.4] - 2026-06-10
- **重构**:
  - 使用 `jsmod.Get` 进行上下文配置提取。

## [1.5.3] - 2026-06-08
- **新增**: `SetConfig(name, setting string)` 方法，支持动态配置数据库连接（不依赖配置文件），方便通过别名获取连接。
- **优化**: 重构配置加载逻辑，支持动态配置与 `config/db.yaml` 配置文件配置共存。

## [1.5.2] - 2026-06-05
- **架构重构: 两阶段 (Two-Pass) 索引命名生成**:
  - 重构了 `CheckTable` 中的 Schema 同步逻辑，从“边解析边生成”改为两阶段处理。现在多列联合索引（无论是 Unique, Index 还是 Fulltext）的名字会根据包含的**字段名称自动拼接生成**（如 `uk_user_name_phone`），替代了以前难以溯源的分组号（如 `uk_user_1`），极大提升了 DBA 在数据库维护时的可读性与直观性。
  - **超长防断机制**: 当生成的索引名称超过 60 个字符时，自动进行截断，并在末尾追加原完整名称的 6 位 MD5 Hash，完美规避了各种 RDBMS 对标识符 64 字符长度限制引发的建表错误。
- **稳定性修复: 幂等性增强**:
  - 为所有支持的数据库（特别是 SQLite 和 PostgreSQL）的 `CREATE INDEX` 及 `CREATE UNIQUE INDEX` 语句增加了 `IF NOT EXISTS` 容错保护。
  - **SQLite 特效探测**: 修复了对 SQLite 多列索引结构探测的 Bug，现在它能准确匹配基于列组隐式创建的索引（如 `sqlite_autoindex`），并将其视作等效的已存在约束跳过重建。彻底根除了在系统重启及结构同步期间偶现的 `UNIQUE constraint failed` 致命崩溃。

## [1.5.0] - 2026-05-10
- **基础设施对齐**: 全局对齐至 v1.5.0。

## [1.3.5] - 2026-05-30
- **新增**: 注册到 `jsmod`。
- **安全性**: 引入基于 Context 的细粒度权限控制。在 `SafeMode` 下，仅允许读取操作（Query），所有写操作（Exec/Insert/Update/Delete/Replace）将被拦截并返回错误。严禁通过 JS 动态创建连接或同步 Schema。

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
