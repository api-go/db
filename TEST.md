# @go/db 测试报告

## 📊 概览
- **模块**: `apigo.cc/go/db`
- **总测试用例**: 5
- **通过**: 5
- **失败**: 0
- **编译状态**: 成功 (Success)
- **测试日期**: 2026-05-03

## ✅ 详细详情
| 测试用例 | 状态 | 耗时 | 备注 |
| :--- | :--- | :--- | :--- |
| `TestMakeInsertSql` | 通过 | 0.00s | 验证 Struct 模型的 SQL 生成逻辑 |
| `TestBaseSelect` | 通过 | 0.00s | 验证结果绑定 (Struct, Map, 基础类型) |
| `TestInsertReplaceUpdateDelete` | 通过 | 0.01s | 验证 SQLite 下的 CRUD 基本操作 |
| `TestTransaction` | 通过 | 0.03s | 验证事务隔离、回滚与提交 |
| `TestSchemaSync` | 通过 | 0.01s | 验证 DSL 同步、影子删除、版本号乐观锁及泛型 API |
| `TestAutoRandomID` | 通过 | 0.01s | 验证 char(N) 主键的自动 ID 填充 |

## 🚀 性能基准 (Benchmarks)
| 基准测试 | 迭代次数 | 耗时 | 内存分配 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| `BenchmarkForPool` | 172009 | 7384 ns/op | 1224 B/op (34 allocs) | 验证 SQLite 下的查询绑定性能 |
| `BenchmarkForPoolParallel` | 160250 | 6852 ns/op | 1296 B/op (35 allocs) | 验证高并发下的查询稳定性 |

## 🛠 环境
- **OS**: darwin (macOS)
- **Go Version**: 1.2x+
- **Primary Driver**: modernc.org/sqlite
