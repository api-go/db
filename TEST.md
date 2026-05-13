# @go/db 测试报告

## 📊 概览
- **模块**: `apigo.cc/go/db`
- **总测试用例**: 7
- **通过**: 7
- **失败**: 0
- **编译状态**: 成功 (Success)
- **测试日期**: 2026-05-13

## ✅ 详细详情
| 测试用例 | 状态 | 耗时 | 备注 |
| :--- | :--- | :--- | :--- |
| `TestMakeInsertSql` | 通过 | 0.00s | 验证 Struct 模型的 SQL 生成逻辑 |
| `TestBaseSelect` | 通过 | 0.00s | 验证结果绑定 (Struct, Map, 基础类型) |
| `TestInsertReplaceUpdateDelete` | 通过 | 0.01s | 验证 SQLite 下s CRUD 基本操作 |
| `TestTransaction` | 通过 | 0.03s | 验证事务隔离、回滚与提交 |
| `TestAutonomousFTS` | 通过 | 0.01s | 验证多种引用风格下的 FTS 重定向 |
| `TestComplexIdentifierFTS` | 通过 | 0.01s | 验证带横杠和表前缀的复杂标识符 FTS 重定向 |
| `TestSchemaSync` | 通过 | 0.01s | 验证 DSL 同步、影子删除、版本号乐观锁及泛型 API |
| `TestAutoRandomID` | 通过 | 0.01s | 验证 char(N) 主键的自动 ID 填充 |

## 🚀 性能基准 (Benchmarks)
| 基准测试 | 迭代次数 | 耗时 | 内存分配 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| `BenchmarkForPool` | 103951 | 11821 ns/op | 1356 B/op (37 allocs) | 增加了复杂标识符解析开销 |
| `BenchmarkForPoolParallel` | 84481 | 13904 ns/op | 1681 B/op (39 allocs) | 验证高并发下的查询稳定性 |

## 🛠 环境
- **OS**: darwin (macOS)
- **Go Version**: 1.25.0
- **Primary Driver**: modernc.org/sqlite
