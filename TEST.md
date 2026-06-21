# @go/db 测试报告

## 📊 概览
- **模块**: `apigo.cc/go/db`
- **总测试用例**: 13
- **通过**: 13
- **失败**: 0
- **编译状态**: 成功 (Success)
- **测试日期**: 2026-06-21

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
| `BenchmarkForPool` | 106807 | 12230 ns/op | - | 验证 SQLite 下的查询绑定性能 (v1.0.11) |
| `BenchmarkForPoolParallel` | 86833 | 15723 ns/op | - | 验证高并发下的查询稳定性 (v1.0.11) |

## 🛠 环境
- **OS**: darwin (macOS)
- **Go Version**: 1.25.0
- **Primary Driver**: modernc.org/sqlite
