# AI 指南 - @go/db

## 🤖 AI 调用规则
- **版本**: v1.0.1
- **核心原则**: 优先使用结构化绑定（`To`, `MapResults`），避免手动拼装 SQL 结果。
- **敏感数据**: 必须通过 `SetEncryptKeys` 配置密钥，确保 DSN 中的密码安全。
- **读写分离**: 鼓励在 DSN 中配置多个 Host 以利用内置的读写分离机制。
- **性能优化**: 
  - 大规模查询应优先绑定到 Struct 切片。
  - 频繁执行的 SQL 应使用 `Prepare`。
- **事务处理**: 始终使用 `tx.Finish(err == nil)` 或 `defer tx.CheckFinished()` 确保事务闭环。

## ⚠️ 注意事项
- 严禁在代码中硬编码数据库凭据。
- 严禁忽略 `Exec` 或 `Query` 返回的 `Error`。
- SQLite 模式下，时间字段会自动转换，无需手动解析字符串。
