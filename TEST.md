# @go/db 测试报告

## 2026-09-04 Schema 事务
- TestSchemaTransactionRollbackAndCommit 验证同行修改、列/索引/影子表结构的事务同步，失败回滚及成功提交；全量和定向 race 通过。

## Unreleased 独立索引
- TestStandaloneIndexes：SQLite AB/BA、混合方向、单列降序、唯一约束、重复同步、索引删除及影子表不继承约束。
- TestStandaloneIndexValidationBeforeMutation：字段引用、重复列、无效标记、全文降序的拒绝；非法索引不能先添加字段。
- 字段类型必须提供；全文用例显式定义 `title t` 并另写 `title TI`，验证独立全文声明，不再把 `title TI` 当字段定义。
- 非 SQLite 实例的物理同步尚未实测；旧数字分组仍待上层迁移后移除，不能视为整个索引改造完成。

## v1.5.14 验证
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
- **模块**: `apigo.cc/go/db`
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
