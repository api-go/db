# @go/db 测试报告

## v1.5.23 发布验证（2026-10-23）

- 使用 `GOWORK=off` 与 go.mod 已发布依赖，顺序执行完整 `go test -v ./...`、`go vet ./...`、`go test -race ./...`、`go test -bench=. ./...`；设置 `DB_TEST_POSTGRES_DSN=pgx:///apigo` 执行真实 PostgreSQL 用例。最终结果与基准见本节下方，原始日志 `/tmp/work-release-20261023-db/`。
- `TestPostgresSchemaIndexTransitions`、`TestPostgresLongSchemaIndexTransitions` 覆盖 Unique→Index→无索引、55字节表名、反向唯一约束失败回滚、重复同步及数据留存；主键、约束支撑、外部普通索引与精确生成名称的 partial 索引保留。独立表均通过原生 DROP 清理。
- `TestQueryFixedCharacterPaddingDoesNotCreatePhantomReferences` 覆盖 CHAR/BPCHAR 空引用、前导空白、VARCHAR/TEXT 尾空格及 map/struct 类型读回。
- `TestQueryJSONByteStorageRetainsObjectsAndOrdinaryBinary` 覆盖 JSON/JSONB 对象/数组的字符串及字节输入、interface 字段、显式字节 map/slice、普通 BLOB、无效 JSON 与 NULL。
- API 工具未提供，使用 `go doc -all github.com/api-go/db` 与导出源码/README 交叉审计；本次没有新增或修改公开声明。apigo 依赖与索引已发布版本一致，无本地 replace。
- 本次完整 test/vet/race/bench 退出码均为 0；原生 PostgreSQL 索引用例未跳过。基准结果（darwin/arm64，Apple M3 Max）：

```text
BenchmarkForPool-16            	  257647	      4381 ns/op
```

- PostgreSQL 上层台账已完成五阶段真实验证：原记录更正、跨对话新增、重复发送及陈旧 CSV 核对；账面结果不等于实物或长期经营效果。

## Unreleased — mixed SQLite values (2026-10-07)

- TestQueryDynamicStorageClasses covers maps, slices, interface struct fields, scalar interfaces, NULL and explicit float destinations. The original scanner fails on integer followed by 0.125; the fixed scanner passes.
- Focused TestQueryDynamicStorageClasses and TestGenericQuery passed; no full suite or external database integration claimed.
- 2026-10-08：`GOWORK=off go test . -run '^(TestQueryDynamicStorageClasses|TestGenericQuery)$' -count=1` passed (0.875s). Local commit only; no release tag, push, full release suite or benchmarks in this handoff.


## v1.5.21 验证（2026-10-06）
- 环境：Apple M3 Max，darwin/arm64；所有命令使用 GOWORK=off，测试实际发布依赖。
- TestSQLiteFTSStringPrimaryKeyAndLegacyTriggerMigration 覆盖字符串主键 CRUD、别名全文查询、旧触发器迁移与回填、重复同步、全文字段替换/删除以及事务回滚保留检索。
- go test -v ./... 通过（db 0.689s）；go test -bench=. ./... 通过（db 2.819s）。
- go test -race ./... 通过（db 2.049s）；go vet ./... 通过。
- BenchmarkForPool-16：4547 ns/op；BenchmarkForPoolParallel-16：8214 ns/op。
- mysql/pgx 驱动包编译通过，但本次未执行外部 MySQL/PostgreSQL 服务集成；本次 Schema 行为变更仅针对 SQLite。
- README API 使用 go doc 与导出源码核对；当前工作区未提供 ai/tools/getExports。

## v1.5.20 公共默认语义
- 覆盖裸 MySQL 不选择数据库、裸 PostgreSQL 使用 `postgres`，以及 `mysql:///apigo`、`pgx:///apigo` 的显式库解析和自动创建。

## v1.5.19 默认库与 MariaDB TCP 引导
- `postgres` 与 `pgx` 真实验证默认创建并连接 `127.0.0.1:5432/apigo`。
- MariaDB 12.3 在仅有 Homebrew 默认 socket 管理员的初始状态下，自动创建 `apigo@localhost/127.0.0.1`、授权 `apigo` 库，随后 TCP 查询成功。
- MySQL/MariaDB 运行连接不依赖 Unix socket；socket 仅用于本机零配置首次引导。

## v1.5.18 PostgreSQL/MySQL 验证
- 单元测试覆盖网络数据库简写、默认 host/端口/库/用户和 PostgreSQL SQL/Schema 转换。
- PostgreSQL 18.6：真实 TCP 零配置连接及不存在数据库自动创建通过。
- MySQL 26.7：`mysql` 使用 root 空密码经本机 TCP 连接、自动创建 `apigo` 并查询通过。
- MariaDB 12.3：默认 TCP root 被 unix_socket 插件拒绝后，受控回退当前系统用户 Unix socket、自动创建 `apigo` 并查询通过。
- `go test -v ./...`、`go test -bench=. ./...`、`go vet ./...` 与 `git diff --check` 通过。

## Unreleased IPv6 参数绑定
- `TestMakeKeysVarsValuesBindsIPv6Loopback` 验证 `::1` 始终作为参数值绑定，不会被解释为 SQL 表达式。

## 2026-09-04 Schema 事务
- TestSchemaTransactionRollbackAndCommit 验证同行修改、列/索引/影子表结构的事务同步，失败回滚及成功提交；全量和定向 race 通过。

## Unreleased 独立索引
- TestStandaloneIndexes：SQLite AB/BA、混合方向、单列降序、唯一约束、重复同步、索引删除及影子表不继承约束。
- TestStandaloneIndexValidationBeforeMutation：字段引用、重复列、无效标记、全文降序的拒绝；非法索引不能先添加字段。
- 字段类型必须提供；全文用例显式定义 `title t` 并另写 `title TI`，验证独立全文声明，不再把 `title TI` 当字段定义。
- 非 SQLite 实例的物理同步尚未实测；旧数字分组仍待上层迁移后移除，不能视为整个索引改造完成。

## v1.5.15 验证
- `TestNumericIndexGroupsAreRejected` 验证 Schema DSL 拒绝 `I<n>`/`U<n>`/`TI<n>` 旧数字分组，复合索引必须使用独立表级声明。
- `go test -v ./...` 与 `go test -bench=. ./...` 作为发布前完整验证执行。
- `git diff --check` 通过。

## v1.5.11 验证
- `TestSchemaSync` 覆盖 SQLite 主表与影子删除表的字段移除。
- `go test -v ./...` 与 `go test -bench=. ./...` 通过。

## v1.5.9 验证
- `go list -m all`：模块图使用 `redis v1.5.11` 与稳定版 `redigo v1.9.3`。
- `go test -v ./...`：通过完整测试。
- `go test -bench=. ./...`：通过完整基准测试。

## 📊 概览
- **模块**: `github.com/api-go/db`
- **总测试用例**: 13
- **通过**: 13
- **失败**: 0
- **编译状态**: 成功 (Success)
- **测试日期**: 2026-07-18

## ✅ 详细详情
| 测试用例 | 状态 | 耗时 | 备注 |
| :--- | :--- | :--- | :--- |
| `TestTableProbing` | 通过 | 0.00s | 验证表结构探测 |
| `TestMakeInsertSql` | 通过 | 0.00s | 验证 Struct 模型的 SQL 生成逻辑 |
| `TestBaseSelect` | 通过 | 0.00s | 验证结果绑定 (Struct, Map, 基础类型) |
| `TestInsertReplaceUpdateDelete` | 通过 | 0.01s | 验证 SQLite 下的 CRUD 基本操作 |
| `TestTransaction` | 通过 | 0.03s | 验证事务隔离、回滚与提交 |
| `TestAutonomousFTS` | 通过 | 0.01s | 验证全文搜索功能 |
| `TestSchemaSync` | 通过 | 0.01s | 验证 DSL 同步、影子删除、版本号乐观锁及泛型 API |
| `TestAutoDetectShadow` | 通过 | 0.00s | 验证影子表自动检测 |
| `TestSmartDelete` | 通过 | 0.01s | 验证智能删除 (物理/影子) |
| `TestGenericQuery` | 通过 | 0.00s | 验证泛型查询映射 |
| `TestAutoRandomID` | 通过 | 0.01s | 验证 char(N) 主键的自动 ID 填充 |
| `TestVersionControl` | 通过 | 0.00s | 验证版本控制递增 |

## 🚀 性能基准 (Benchmarks)
| 基准测试 | 迭代次数 | 耗时 | 内存分配 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| `BenchmarkForPool` | 245385 | 4866 ns/op | - | 验证 SQLite 下的查询绑定性能 (v1.5.9) |
| `BenchmarkForPoolParallel` | 136383 | 8838 ns/op | - | 验证高并发下的查询稳定性 (v1.5.9) |

## 🛠 环境
- **OS**: darwin (macOS)
- **Go Version**: 1.26.4
- **Primary Driver**: modernc.org/sqlite

Unreleased datetime regression: declared DATETIME values retain time and fractional seconds; DATE remains date-only. Interface SQL temporal values retain the existing textual contract. TestQueryDateTimePreservesTime fails with the old date-first matching and passes with datetime-first normalization.

## v1.5.22 发布验证（2026-10-10）

- GOWORK=off go test -v ./... 与 go test -bench=. ./... 通过，依赖均为已发布 tag。
- API 以 GOWORK=off go doc 与导出源码交叉核对（getExports 工具不可用）。

```text
BenchmarkForPool-16            	  254239	      4721 ns/op
BenchmarkForPoolParallel-16    	# connection count 0 0 {"maxOpenConnections":100,"openConnections":2,"inUse":0,"idle":2,"waitCount":0,"waitDuration":0,"maxIdleClosed":17,"maxIdleTimeClosed":0,"maxLifetimeClosed":0} .
```

## v1.5.24 发布验证（2026-10-26）

GOWORK=off，正式依赖 crypto v1.5.9，无 replace。`go test -v ./...`、`go test -bench=. ./...`、`go vet ./...`、`go mod verify` 通过。使用 `go doc` 与源码交叉核对 API，替代 getExports。现有回归覆盖本模块业务；新旧 GCM 格式、认证失败和性能基准见 crypto v1.5.9。

跳过的外部条件场景：
- --- SKIP: TestPostgresSchemaIndexTransitions (0.00s)
- --- SKIP: TestPostgresLongSchemaIndexTransitions (0.00s)

基准结果：
```text
BenchmarkForPool-16            	  249373	      4820 ns/op
BenchmarkForPoolParallel-16    	# connection count 0 0 {"maxOpenConnections":100,"openConnections":2,"inUse":0,"idle":2,"waitCount":0,"waitDuration":0,"maxIdleClosed":14,"maxIdleTimeClosed":0,"maxLifetimeClosed":0} .
```

## GitHub migration validation (pending release)

Module paths and internal dependency versions are migrated. Development verification uses GOWORK=off and explicit sibling replacements; remove these replacements before release. See migration validation logs for actual results. Existing unrelated working changes are preserved and require separate review.

API inventory verification: `GOWORK=off go doc` succeeded; output retained in the migration validation directory. Source exports were preserved by the path migration.

## GitHub release verification

GOWORK=off go test -v ./... and go test -bench=. ./... passed using published GitHub dependencies, without local replacements. go mod verify passed.

## 日志依赖联动 — 2026-11-12

开发验证使用 GOWORK=off 与 go.mod 的显式本地 replace；执行完整测试、全量基准（benchtime=100ms），API 用 go doc 与导出源码交叉核对。结果记录在 /tmp/apigo-db-integration-*.txt。

结果：完整测试与全量基准通过（本机，GOWORK=off）；外部依赖不具备时的测试跳过以原始输出为准。


## v1.5.26 发布验证 — 2026-11-12

GOWORK=off；正式 GitHub 依赖，无本地 replace。完整 go test -v ./... 与 go test -bench=. -benchtime=100ms ./... 通过；Service/Gateway 设置 GODEBUG=http2xconnect=1。API 核对使用 go doc . 与导出源码（getExports 工具不可用）。可选外部环境测试按原规则跳过。详细输出 /tmp/apigo-db-release-tests.txt 与 /tmp/apigo-db-release-bench.txt。
