# @go/db

> **Maintainer Statement:** 本项目由 AI 维护。代码源自 github.com/ssgo/db 的重构，支持内存安全防护、读写分离及泛型优化。

## 🎯 设计哲学

`@go/db` 是一个极致精简、意图优先的数据库抽象层。它不试图取代 SQL，而是通过智能结果绑定与 SQL 自动化生成，消除数据库操作中的样板代码。

*   **智能绑定**：根据结果容器类型（Struct/Map/Slice/BaseType）自动适配查询逻辑，无需手动 Scan。
*   **内存防御**：集成 `go/safe`，数据库密码在内存中加密存储，使用时物理擦除。
*   **读写分离**：内置连接池管理，支持配置多个只读节点实现自动负载均衡。
*   **驱动透明**：统一 MySQL、PostgreSQL (pgx) 与 SQLite 的 API 差异。

## 📦 安装

```bash
go get apigo.cc/go/db
```

## 💡 快速开始

```go
import "apigo.cc/go/db"
import _ "apigo.cc/go/db/mysql" // 引入驱动

// 初始化连接
d := db.GetDB("mysql://user:pass@host:3306/dbname", nil)

// 1. 查询全部结果到 Struct 切片
var users []User
d.Query("SELECT * FROM users").To(&users)

// 2. 自动化插入
d.Insert("users", User{Name: "Star", Active: true})

// 3. 事务操作
tx := d.Begin()
tx.Exec("UPDATE balance SET amount = amount - 10 WHERE id = ?", 1)
tx.Commit()
```

## 🛠 API 指南

### 核心对象
- **`GetDB(setting string, logger *log.Logger) *DB`**: 通过 DSN 或配置名获取数据库实例。
- **`DB.Insert/Replace/Update/Delete`**: 自动生成 SQL 并执行，支持 Struct 与 Map。
- **`QueryResult.To(target any)`**: 将查询结果深度映射到目标容器。
- **`QueryResult.MapResults() []map[string]any`**: 快捷获取通用结果集。

### 结果容器适配规则
| 容器类型 | 行为 |
| :--- | :--- |
| `[]Struct` | 返回所有行，按字段名自动映射 |
| `Struct` | 返回第一行，按字段名自动映射 |
| `[]map[string]any` | 返回所有行，保留原始字段名 |
| `[]BaseType` | 返回所有行，仅取第一列 |
| `BaseType` | 返回第一行第一列 |

### 安全与高级特性
- **`SetEncryptKeys(key, iv []byte)`**: 配置全局敏感数据加密密钥。
- **读写分离**: 在 DSN 中配置 `host1,host2,host3`，首个为主库，其余为随机只读库。
- **SQLite 时间修复**: 自动处理 SQLite 毫秒级 `DATETIME` 格式与标准 `time.Time` 的转换。

## 🧪 验证状态
已通过 SQLite 集成测试。详见：[TEST.md](./TEST.md)
