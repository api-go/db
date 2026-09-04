# 数据库架构 DSL (Schema-as-Code)

本模块提供了一种基于文本的 DSL（领域专用语言）来定义数据库架构。它支持 MySQL、PostgreSQL 和 SQLite，旨在实现 AI 友好的“架构即代码”。

## 语法概览

一个架构描述由 **分组 (Groups)**、**数据表 (Tables)** 和 **字段 (Fields)** 组成。

### 分组 (Groups)
分组由以 `==` 开头的行定义，用于逻辑隔离不同的表集合。
```
== 用户系统 ==
```

### 数据表 (Tables)
数据表在分组下顶格定义。可以在 `//` 后添加注释。
在表名后添加 `SD` 标记可启用 **影子删除 (Shadow Deletion)**。启用后，删除的数据会自动移动到 `[表名]_deleted` 表中。
```
users SD // 系统用户表
```

### 字段 (Fields)
字段在数据表下通过缩进（空格或制表符）定义。
格式：`[名称] [类型标记][长度] [索引标记] // [注释]`

```
  id i AI // 主键，自动递增
  username v32 U // Varchar(32)，唯一索引
  password v64 // Varchar(64)
  version ver // Bigint 版本号，用于乐观锁和增量同步
  create_time dt ct // 创建时间 (CURRENT_TIMESTAMP)
  update_time dt ctu // 更新时间 (ON UPDATE CURRENT_TIMESTAMP)
```

## 类型标记 (Type Tags)

| 标记 | 对应数据库类型 (MySQL) | 说明 |
|-----|-----------------|------|
| `i` | `int` | 整型 |
| `ui`| `int unsigned` | 无符号整型 |
| `bi`| `bigint` | 长整型 |
| `ubi`| `bigint unsigned` | 无符号长整型 |
| `ti`| `tinyint` | 短整型 |
| `v` | `varchar` | 默认长度由驱动决定或忽略 |
| `v[N]`| `varchar(N)` | 例如：`v50` -> `varchar(50)` |
| `c[N]`| `char(N)` | 例如：`c32` -> `char(32)` |
| `t` | `text` | 文本 |
| `dt`| `datetime` | 日期时间 |
| `d` | `date` | 日期 |
| `tm`| `time` | 时间 |
| `f` | `float` | 浮点数 |
| `ff`| `double` | 双精度浮点数 |
| `b` | `tinyint unsigned`| 布尔值别名 |
| `bb`| `blob` | 二进制大对象 |

## 索引与特殊标记

| 标记 | 含义 |
|-----|---------|
| `PK` | 主键 (Primary Key) |
| `AI` | 自动递增 + 主键 (Auto Increment) |
| `U`  | 唯一索引 (Unique Index) |
| `I`  | 普通索引 (Index) |
| `TI` | 全文索引 (Fulltext Index, 仅 MySQL) |
| `ver`| 版本号字段 (用于乐观锁和增量同步) |
| `ct` | 创建时间 (Created Time) |
| `ctu`| 更新时间 (Updated Time) |
| `nn` | 非空 (NOT NULL) |
| `n`  | 可为空 (NULL) |

### 独立索引
字段类型必须提供；字段后的 I/U/TI 仅用于单字段升序索引。独立索引行不写类型，只写有序字段清单和 I/U/TI；支持单字段。前缀 `-` 表示降序，其余为升序，不使用 ASC/DESC 关键字。

```
users
  id c12 PK
  username v32
  email v64
  status ti I
  updated_at bi
  username,email U
  email,username I
  status,-updated_at I
  -updated_at I
  username I
```

名称由表名、类型及有序字段/方向自动生成。字段不存在、重复引用字段或全文索引指定降序时拒绝同步。Sync 在执行任何表的 DDL 前校验全部表；删除声明会移除对应索引。主键仍写在字段定义上。

Go 入口为 `TableStruct.Indexes []TableIndex`，例如 `TableIndex{Type: "I", Fields: []string{"status", "-updated_at"}}`。旧数字分组仅在 TableDB 联动迁移期间暂留，不用于新定义。

## 高级特性

### 1. 影子删除 (SD - Shadow Deletion)
当表标记为 `SD` 时，调用 `db.Remove()` 方法不会真正删除数据，而是将其从原表移动到 `_deleted` 后缀的影子表中。
- **优点**：主表查询不包含已删除数据，效率更高；历史数据可追溯。
- **API**: 使用 `db.Remove(table, conditions, args...)` 触发。

### 2. 乐观锁与增量同步 (ver)
标记为 `ver` 的字段（通常命名为 `version`）具有特殊行为：
- **自动递增**：每次调用 `db.Update()` 时，该字段会自动 `+1`。
- **冲突检测**：如果在更新数据中包含了当前版本号，`db.Update()` 会在 `WHERE` 条件中自动加入版本校验。如果版本不一致，更新将失败（影响行数为 0）。
- **增量同步**：外部系统可以通过 `WHERE version > last_version` 轻松获取自上次同步以来的所有变更。

## 完整示例
```
== 默认分组 ==
users SD // 用户表
  id i AI // 用户 ID
  username v32 U // 登录名
  email v64 U // 联系邮箱
  password v128 // 加密后的密码
  status ti I // 0: 活跃, 1: 禁用
  version ver // 行版本号
  created_at dt ct // 创建记录
  updated_at dt ctu // 更新记录
```
