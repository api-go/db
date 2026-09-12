package db

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"apigo.cc/go/cast"
)

// SchemaGroup 描述表分组
type SchemaGroup struct {
	Name   string
	Tables []*TableStruct
}

var fieldSpliter = regexp.MustCompile(`\s+`)
var wnMatcher = regexp.MustCompile(`^([a-zA-Z]+)([0-9]+)$`)

func splitSchemaField(line string) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	if line[0] != '"' {
		return fieldSpliter.Split(line, 10)
	}
	for i, escaped := 1, false; i < len(line); i++ {
		if escaped {
			escaped = false
			continue
		}
		if line[i] == '\\' {
			escaped = true
			continue
		}
		if line[i] == '"' {
			name, err := strconv.Unquote(line[:i+1])
			if err != nil {
				return fieldSpliter.Split(line, 10)
			}
			rest := strings.TrimSpace(line[i+1:])
			parts := []string{name}
			if rest != "" {
				parts = append(parts, fieldSpliter.Split(rest, 9)...)
			}
			return parts
		}
	}
	return fieldSpliter.Split(line, 10)
}

// ParseField 解析单行字段描述
func ParseField(line string) TableField {
	lc := strings.SplitN(line, "//", 2)
	comment := ""
	if len(lc) == 2 {
		line = strings.TrimSpace(lc[0])
		comment = strings.TrimSpace(lc[1])
	}

	a := canonicalFieldParts(splitSchemaField(line))
	if err := validateSchemaField(a); err != nil {
		return TableField{parseError: err}
	}
	field := TableField{
		Name:       a[0],
		Type:       "",
		Index:      "",
		IndexGroup: "",
		Default:    "",
		Comment:    comment,
		Null:       "NULL",
		Extra:      "",
		Desc:       "",
		IsVersion:  false,
	}

	for i := 1; i < len(a); i++ {
		wn := wnMatcher.FindStringSubmatch(a[i])
		tag := a[i]
		size := 0
		if wn != nil {
			tag = wn[1]
			size = cast.Int(wn[2])
		}
		switch tag {
		case "PK":
			field.Index = "pk"
			field.Null = "NOT NULL"
		case "I":
			field.Index = "index"
		case "AI":
			field.Extra = "AUTO_INCREMENT"
			field.Index = "pk"
			field.Null = "NOT NULL"
		case "TI":
			field.Index = "fulltext"
		case "U":
			field.Index = "unique"
		case "ver":
			field.IsVersion = true
			field.Type = "bigint"
			field.Default = "0"
			field.Null = "NOT NULL"
		case "ct":
			field.Default = "CURRENT_TIMESTAMP"
		case "ctu":
			field.Default = "CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"
		case "n":
			field.Null = "NULL"
		case "nn":
			field.Null = "NOT NULL"
		case "c":
			field.Type = "char"
		case "v":
			field.Type = "varchar"
		case "v1", "v2", "v3", "v4", "v5", "v6", "v7", "v8", "v9":
			field.Type = "varchar"
		case "dt":
			field.Type = "datetime"
		case "d":
			field.Type = "date"
		case "tm":
			field.Type = "time"
		case "i":
			field.Type = "int"
		case "ui":
			field.Type = "int unsigned"
		case "ti":
			field.Type = "tinyint"
		case "uti":
			field.Type = "tinyint unsigned"
		case "b":
			field.Type = "boolean"
		case "bi":
			field.Type = "bigint"
		case "ubi":
			field.Type = "bigint unsigned"
		case "f":
			field.Type = "float"
		case "uf":
			field.Type = "float unsigned"
		case "ff":
			field.Type = "double"
		case "uff":
			field.Type = "double unsigned"
		case "si":
			field.Type = "smallint"
		case "usi":
			field.Type = "smallint unsigned"
		case "mi":
			field.Type = "middleint"
		case "umi":
			field.Type = "middleint unsigned"
		case "t":
			field.Type = "text"
		case "o":
			field.Type = "json"
			field.IsObject = true
		case "bb":
			field.Type = "blob"
		default:
			field.Type = tag
		}

		if size > 0 {
			switch tag {
			case "I", "U", "TI":
				field.parseError = fmt.Errorf("numeric index groups are not supported; declare a table-level index")
			default:
				if !field.IsVersion {
					field.Type += fmt.Sprintf("(%d)", size)
				}
			}
		}
	}
	return field
}

// ParseSchema 解析完整的 Schema 描述文本
func ParseSchema(desc string) []*SchemaGroup {
	groups := make([]*SchemaGroup, 0)
	var currentGroup *SchemaGroup
	var currentTable *TableStruct

	lines := strings.Split(desc, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		if strings.HasPrefix(trimmedLine, "==") {
			currentGroup = &SchemaGroup{
				Name:   strings.TrimSpace(strings.Trim(trimmedLine, "=")),
				Tables: make([]*TableStruct, 0),
			}
			groups = append(groups, currentGroup)
			continue
		}

		if currentGroup == nil {
			currentGroup = &SchemaGroup{
				Name:   "Default",
				Tables: make([]*TableStruct, 0),
			}
			groups = append(groups, currentGroup)
		}

		if !strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "\t") {
			lc := strings.SplitN(trimmedLine, "//", 2)
			tableNamePart := strings.TrimSpace(lc[0])
			comment := ""
			if len(lc) == 2 {
				comment = strings.TrimSpace(lc[1])
			}

			shadowDelete := false
			if strings.HasSuffix(tableNamePart, " SD") {
				shadowDelete = true
				tableNamePart = strings.TrimSpace(strings.TrimSuffix(tableNamePart, " SD"))
			}

			currentTable = &TableStruct{
				Name:         tableNamePart,
				Comment:      comment,
				Fields:       make([]TableField, 0),
				ShadowDelete: shadowDelete,
			}
			currentGroup.Tables = append(currentGroup.Tables, currentTable)
		} else if currentTable != nil {
			if index, matched, err := parseTableIndex(trimmedLine); matched {
				if err != nil {
					currentTable.indexError = err
				} else {
					currentTable.Indexes = append(currentTable.Indexes, index)
				}
				continue
			}
			field := ParseField(trimmedLine)
			if field.IsVersion {
				currentTable.VersionField = field.Name
			}
			if field.Index == "fulltext" {
				// 保持原字段，但移除其索引标记，由影子列承担索引
				field.Index = ""
				currentTable.Fields = append(currentTable.Fields, field)

				// 隐式追加影子列
				tokensField := TableField{
					Name:    field.Name + "_tokens",
					Type:    "text",
					Null:    "NULL",
					Index:   "fulltext",
					Comment: "FTS tokens for " + field.Name,
				}
				currentTable.Fields = append(currentTable.Fields, tokensField)
			} else {
				currentTable.Fields = append(currentTable.Fields, field)
			}
		}
	}
	return groups
}

// Parse 根据数据库类型解析字段的基础描述
func (field *TableField) Parse(tableType string) {
	if strings.HasPrefix(tableType, "sqlite") || tableType == "chai" {
		// sqlite3 不能修改字段，统一使用NULL
		field.Null = "NULL"
		if field.Extra == "AUTO_INCREMENT" || field.Extra == "AUTOINCREMENT" {
			field.Extra = "PRIMARY KEY AUTOINCREMENT"
			field.Type = "integer"
			field.Null = "NOT NULL"
		}
	}

	a := make([]string, 0)
	if tableType == "mysql" {
		a = append(a, fmt.Sprintf("`%s` %s", field.Name, field.Type))
		lowerType := strings.ToLower(field.Type)
		if strings.Contains(lowerType, "varchar") || strings.Contains(lowerType, "text") {
			a = append(a, " COLLATE utf8mb4_general_ci")
		}
	} else if isPostgresType(tableType) {
		typ := postgresFieldType(field.Type)
		if field.Extra == "AUTO_INCREMENT" {
			if strings.Contains(typ, "bigint") {
				typ = "bigserial"
			} else {
				typ = "serial"
			}
			field.Extra = ""
		}
		a = append(a, fmt.Sprintf("\"%s\" %s", field.Name, typ))
	} else {
		typ := field.Type
		if strings.HasPrefix(tableType, "sqlite") && strings.EqualFold(typ, "boolean") {
			// Keep SQLite's historical numeric affinity and result conversion for b.
			typ = "tinyint unsigned"
		}
		a = append(a, fmt.Sprintf("\"%s\" %s", field.Name, typ))
	}

	if field.Extra != "" && !strings.Contains(field.Extra, "PRIMARY KEY") {
		a = append(a, " "+field.Extra)
	}
	a = append(a, " "+field.Null)

	if field.Default != "" {
		if strings.Contains(field.Default, "CURRENT_TIMESTAMP") || strings.Contains(field.Default, "()") || strings.Contains(field.Default, "SYSTIMESTAMP") {
			a = append(a, " DEFAULT "+field.Default)
		} else {
			a = append(a, " DEFAULT '"+field.Default+"'")
		}
	}

	if strings.HasPrefix(tableType, "sqlite") || tableType == "chai" {
		field.Comment = ""
		field.Type = "numeric"
	} else if isPostgresType(tableType) {
		// PostgreSQL comments are separate statements
	} else {
		if field.Comment != "" {
			a = append(a, " COMMENT '"+field.Comment+"'")
		}
	}
	field.Desc = strings.Join(a, "")
}

func isPostgresType(tableType string) bool {
	switch tableType {
	case "pg", "pgsql", "postgres", "postgresql", "pgx":
		return true
	default:
		return false
	}
}

func postgresFieldType(fieldType string) string {
	lower := strings.ToLower(strings.TrimSpace(fieldType))
	lower = strings.TrimSuffix(lower, " unsigned")
	switch lower {
	case "tinyint", "smallint":
		return "smallint"
	case "middleint", "int", "integer":
		return "integer"
	case "bigint":
		return "bigint"
	case "float":
		return "real"
	case "double":
		return "double precision"
	case "datetime":
		return "timestamp"
	case "blob":
		return "bytea"
	case "json":
		return "jsonb"
	default:
		return fieldType
	}
}

// Sync 根据 Schema 描述文本同步数据库结构
func (db *DB) Sync(desc string) error {
	groups := ParseSchema(desc)
	// Validate every table before changing any physical schema.
	for _, group := range groups {
		for _, table := range group.Tables {
			if err := table.prepareIndexes(); err != nil {
				return err
			}
		}
	}
	var outErr error
	for _, group := range groups {
		for _, table := range group.Tables {
			db.tablesLock.Lock()
			delete(db.tables, table.Name)
			db.tablesLock.Unlock()

			err := db.CheckTable(table)
			if err != nil {
				outErr = err
				db.logger.logger.Error("failed to sync table", "table", table.Name, "err", err.Error())
			}
		}
	}
	return outErr
}

// CheckTable 检查并同步单个表结构
func (db *DB) CheckTable(table *TableStruct) error {
	if err := table.prepareIndexes(); err != nil {
		return err
	}
	fieldSets := make([]string, 0)
	pks := make([]string, 0)
	keySets := make([]string, 0)
	keySetBy := make(map[string]string)
	keySetFields := make(map[string]string)
	ftsFields := make([]string, 0)

	isPostgres := isPostgresType(db.Config.Type)

	table.Columns = make([]string, 0, len(table.Fields))

	// 阶段一：收集所有属于同一索引组的字段
	type indexGroupDef struct {
		Type   string
		Fields []string
	}
	indexGroups := make(map[string]*indexGroupDef)

	for i, field := range table.Fields {
		field.Parse(db.Config.Type)
		table.Fields[i] = field
		table.Columns = append(table.Columns, field.Name)

		idxType := strings.ToLower(field.Index)
		if idxType == "pk" || idxType == "primary key" {
			pks = append(pks, field.Name)
		} else if idxType == "unique" || idxType == "index" || idxType == "fulltext" {
			group := field.IndexGroup
			if group == "" {
				group = field.Name
			}
			groupKey := idxType + "_" + group
			if _, exists := indexGroups[groupKey]; !exists {
				indexGroups[groupKey] = &indexGroupDef{Type: idxType, Fields: []string{}}
			}
			indexGroups[groupKey].Fields = append(indexGroups[groupKey].Fields, field.Name)
		}
	}

	// 阶段二：生成字段定义与索引定义
	processedGroups := make(map[string]bool)

	indexFields := append([]TableField{}, table.Fields...)
	for i, index := range table.Indexes {
		kind := map[string]string{"I": "index", "U": "unique", "TI": "fulltext"}[strings.ToUpper(index.Type)]
		group := fmt.Sprintf("@table_%d", i)
		columns := append([]string{}, index.Fields...)
		if kind == "fulltext" {
			for j := range columns {
				columns[j] += "_tokens"
			}
		}
		indexGroups[kind+"_"+group] = &indexGroupDef{Type: kind, Fields: columns}
		indexFields = append(indexFields, TableField{Index: kind, IndexGroup: group})
	}
	for _, field := range indexFields {
		idxType := strings.ToLower(field.Index)
		if idxType == "unique" || idxType == "index" || idxType == "fulltext" {
			group := field.IndexGroup
			if group == "" {
				group = field.Name
			}
			groupKey := idxType + "_" + group

			if !processedGroups[groupKey] {
				processedGroups[groupKey] = true
				groupDef := indexGroups[groupKey]

				// 根据组合的字段名拼接生成长索引名
				joinedFields := strings.Join(groupDef.Fields, "_")
				prefix := "ik"
				if idxType == "unique" {
					prefix = "uk"
				} else if idxType == "fulltext" {
					prefix = "tk"
				}
				keyName := fmt.Sprintf("%s_%s_%s", prefix, table.Name, joinedFields)
				if strings.HasPrefix(group, "@table_") {
					// Include boundaries and directions; a_b,c must not collide
					// with a,b_c, nor ASC with DESC or another index type.
					hash := md5.Sum([]byte(idxType + "\x00" + strings.Join(groupDef.Fields, "\x00")))
					keyName = fmt.Sprintf("%s_%s_%s", prefix, table.Name, hex.EncodeToString(hash[:])[:16])
				}

				// 防超长截断处理 (最大 64，留 4 个给安全余量，设为 60)
				if len(keyName) > 60 {
					hash := md5.Sum([]byte(keyName))
					hashStr := hex.EncodeToString(hash[:])[:6]
					keyName = keyName[:53] + "_" + hashStr
				}

				keySetFields[keyName] = strings.Join(groupDef.Fields, " ")

				if idxType == "fulltext" {
					ftsFields = append(ftsFields, groupDef.Fields...)
					keySet := ""
					if isPostgres {
						// PostgreSQL 使用 simple 分词器
						keySet = fmt.Sprintf("CREATE INDEX IF NOT EXISTS \"%s\" ON \"%s\" USING GIN (to_tsvector('simple', \"%s\"))", keyName, table.Name, groupDef.Fields[0])
					} else if !strings.HasPrefix(db.Config.Type, "sqlite") && db.Config.Type != "chai" {
						keySet = fmt.Sprintf("FULLTEXT KEY "+db.Quote("%s")+" ("+db.indexColumns(groupDef.Fields)+") COMMENT '%s'", keyName, field.Comment)
					}
					if keySet != "" {
						keySets = append(keySets, keySet)
						keySetBy[keyName] = keySet
					}
				} else if idxType == "unique" {
					keySet := ""
					if strings.HasPrefix(db.Config.Type, "sqlite") || db.Config.Type == "chai" {
						keySet = fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS \"%s\" ON \"%s\" (%s)", keyName, table.Name, db.indexColumns(groupDef.Fields))
					} else if isPostgres {
						keySet = fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS \"%s\" ON \"%s\" (%s)", keyName, table.Name, db.indexColumns(groupDef.Fields))
					} else {
						keySet = fmt.Sprintf("UNIQUE KEY "+db.Quote("%s")+" ("+db.mysqlIndexColumns(table, groupDef.Fields)+") COMMENT '%s'", keyName, field.Comment)
					}
					keySets = append(keySets, keySet)
					keySetBy[keyName] = keySet
				} else { // index
					keySet := ""
					if strings.HasPrefix(db.Config.Type, "sqlite") || db.Config.Type == "chai" {
						keySet = fmt.Sprintf("CREATE INDEX IF NOT EXISTS \"%s\" ON \"%s\" (%s)", keyName, table.Name, db.indexColumns(groupDef.Fields))
					} else if isPostgres {
						keySet = fmt.Sprintf("CREATE INDEX IF NOT EXISTS \"%s\" ON \"%s\" (%s)", keyName, table.Name, db.indexColumns(groupDef.Fields))
					} else {
						keySet = fmt.Sprintf("KEY "+db.Quote("%s")+" ("+db.mysqlIndexColumns(table, groupDef.Fields)+") COMMENT '%s'", keyName, field.Comment)
					}
					keySets = append(keySets, keySet)
					keySetBy[keyName] = keySet
				}
			}
		}
		if field.Name != "" {
			fieldSets = append(fieldSets, field.Desc)
		}
	}

	var tableInfo map[string]any
	if strings.HasPrefix(db.Config.Type, "sqlite") {
		tableInfo = db.Query("SELECT \"name\", \"sql\" FROM \"sqlite_master\" WHERE \"type\"='table' AND \"name\"='" + table.Name + "'").MapOnR1()
	} else if db.Config.Type == "chai" {
		tableInfo = db.Query("SELECT \"name\", \"sql\" FROM \"__chai_catalog\" WHERE \"type\"='table' AND \"name\"='" + table.Name + "'").MapOnR1()
	} else if isPostgres {
		tableInfo = db.Query("SELECT tablename name FROM pg_tables WHERE schemaname='public' AND tablename='" + table.Name + "'").MapOnR1()
	} else {
		tableInfo = db.Query("SELECT TABLE_NAME name, TABLE_COMMENT comment FROM information_schema.TABLES WHERE TABLE_SCHEMA='" + db.Config.DB + "' AND TABLE_NAME='" + table.Name + "'").MapOnR1()
	}

	if tableInfo["name"] != nil && tableInfo["name"] != "" {
		oldFieldList := make([]*tableFieldDesc, 0)
		oldFields := make(map[string]*tableFieldDesc)
		oldIndexes := make(map[string]string)
		oldUniqueMap := make(map[string]bool)
		oldIndexInfos := make([]*tableKeyDesc, 0)
		oldComments := map[string]string{}

		if strings.HasPrefix(db.Config.Type, "sqlite") {
			tmpFields := []struct {
				Name       string
				Type       string
				Notnull    int
				Dflt_value any
				Pk         int
			}{}
			if err := db.Query("PRAGMA table_info(" + db.Quote(table.Name) + ")").To(&tmpFields); err != nil {
				return err
			}
			for _, f := range tmpFields {
				oldFieldList = append(oldFieldList, &tableFieldDesc{
					Field:   f.Name,
					Type:    f.Type,
					Null:    cast.If(f.Notnull != 0, "NO", "YES"),
					Key:     cast.If(f.Pk != 0, "PRI", ""),
					Default: cast.String(f.Dflt_value),
				})
			}
			tmpIndexes := []struct {
				Name    string
				Unique  int
				Origin  string
				Partial int
			}{}
			if err := db.Query("PRAGMA index_list(" + db.Quote(table.Name) + ")").To(&tmpIndexes); err != nil {
				return err
			}
			for _, i := range tmpIndexes {
				oldUniqueMap[i.Name] = i.Unique == 1
				tmpIndexInfo := []struct {
					Name  string
					Seqno int
					Cid   int
					Desc  int
					Key   int
				}{}
				if err := db.Query("PRAGMA index_xinfo(" + db.Quote(i.Name) + ")").To(&tmpIndexInfo); err != nil {
					return err
				}
				for _, ii := range tmpIndexInfo {
					if ii.Key == 0 {
						continue
					}
					if ii.Desc == 1 {
						ii.Name = "-" + ii.Name
					}
					oldIndexInfos = append(oldIndexInfos, &tableKeyDesc{
						Key_name:    i.Name,
						Column_name: ii.Name,
					})
				}
			}
		} else if isPostgres {
			tmpFields := []struct {
				Column_name    string
				Data_type      string
				Is_nullable    string
				Column_default string
			}{}
			db.Query("SELECT column_name, data_type, is_nullable, column_default FROM information_schema.columns WHERE table_schema='public' AND table_name='" + table.Name + "'").To(&tmpFields)
			for _, f := range tmpFields {
				oldFieldList = append(oldFieldList, &tableFieldDesc{
					Field:   f.Column_name,
					Type:    f.Data_type,
					Null:    f.Is_nullable,
					Default: f.Column_default,
				})
			}
			tmpIndexes := []struct {
				Indexname string
				Indexdef  string
			}{}
			db.Query("SELECT indexname, indexdef FROM pg_indexes WHERE schemaname='public' AND tablename='" + table.Name + "'").To(&tmpIndexes)
			for _, i := range tmpIndexes {
				// Parse indexdef to get columns if needed, simplified for now
				oldIndexes[i.Indexname] = "" // Placeholder
			}
		} else if db.Config.Type == "mysql" {
			_ = db.Query("SELECT column_name, column_comment FROM information_schema.columns WHERE TABLE_SCHEMA='" + db.Config.DB + "' AND TABLE_NAME='" + table.Name + "'").ToKV(&oldComments)
			_ = db.Query("DESC " + db.Quote(table.Name)).To(&oldFieldList)
			_ = db.Query("SHOW INDEX FROM " + db.Quote(table.Name)).To(&oldIndexInfos)
		}

		if !isPostgres {
			for _, indexInfo := range oldIndexInfos {
				if oldIndexes[indexInfo.Key_name] == "" {
					oldIndexes[indexInfo.Key_name] = indexInfo.Column_name
				} else {
					oldIndexes[indexInfo.Key_name] += " " + indexInfo.Column_name
				}
			}
		}

		prevFieldId := ""
		for _, field := range oldFieldList {
			if strings.HasPrefix(db.Config.Type, "sqlite") {
				field.Type = "numeric"
			} else {
				field.After = prevFieldId
			}
			prevFieldId = field.Field
			oldFields[field.Field] = field
		}

		actions := make([]string, 0)
		for keyId := range oldIndexes {
			if keyId != "PRIMARY" && !isPostgres && strings.ToLower(keySetFields[keyId]) != strings.ToLower(oldIndexes[keyId]) {
				if strings.HasPrefix(db.Config.Type, "sqlite") {
					if table.Name == "_Table" || table.Name == "_Field" || table.Name == "_Table_deleted" || table.Name == "_Field_deleted" {
						continue
					}
					if strings.HasPrefix(keyId, "sqlite_autoindex_") {
						continue
					}
					actions = append(actions, "DROP INDEX "+db.Quote(keyId))
				} else {
					actions = append(actions, "DROP KEY "+db.Quote(keyId))
				}
			}
		}

		if oldIndexes["PRIMARY"] != "" && !isPostgres && strings.ToLower(oldIndexes["PRIMARY"]) != strings.ToLower(strings.Join(pks, " ")) {
			if !strings.HasPrefix(db.Config.Type, "sqlite") {
				actions = append(actions, "DROP PRIMARY KEY")
			}
		}

		newFieldExists := map[string]bool{}
		prevFieldId = ""
		for _, field := range table.Fields {
			newFieldExists[field.Name] = true
			oldField := oldFields[field.Name]
			if oldField == nil {
				if strings.HasPrefix(db.Config.Type, "sqlite") {
					actions = append(actions, "ALTER TABLE "+db.Quote(table.Name)+" ADD COLUMN "+field.Desc)
				} else if isPostgres {
					actions = append(actions, "ALTER TABLE "+db.Quote(table.Name)+" ADD COLUMN "+field.Desc)
				} else {
					actions = append(actions, "ADD COLUMN "+field.Desc)
				}
			} else {
				if isPostgres {
					// Postgres sync is more complex, skipped for brevity in this step but should be robust
					continue
				}
				oldField.Type = strings.TrimSpace(strings.ReplaceAll(oldField.Type, " (", "("))
				fixedOldDefault := oldField.Default
				if fixedOldDefault == "CURRENT_TIMESTAMP" && strings.Contains(oldField.Extra, "on update CURRENT_TIMESTAMP") {
					fixedOldDefault = "CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"
				}
				fixedOldNull := "NOT NULL"
				if oldField.Null == "YES" {
					fixedOldNull = "NULL"
				}

				if strings.ToLower(field.Type) != strings.ToLower(oldField.Type) || strings.ToLower(field.Default) != strings.ToLower(fixedOldDefault) || strings.ToLower(field.Null) != strings.ToLower(fixedOldNull) || (db.Config.Type == "mysql" && strings.ToLower(oldField.After) != strings.ToLower(prevFieldId)) || strings.ToLower(oldComments[field.Name]) != strings.ToLower(field.Comment) {
					after := ""
					if db.Config.Type == "mysql" {
						if oldField.After != prevFieldId {
							if prevFieldId == "" {
								after = " FIRST"
							} else {
								after = " AFTER " + db.Quote(prevFieldId)
							}
						}
						actions = append(actions, "CHANGE `"+field.Name+"` "+field.Desc+after)
					}
				}
			}
			if db.Config.Type == "mysql" {
				prevFieldId = field.Name
			}
		}

		for oldFieldName := range oldFields {
			if !newFieldExists[oldFieldName] {
				if strings.HasPrefix(db.Config.Type, "sqlite") {
					if table.Name == "_Table" || table.Name == "_Field" || table.Name == "_Table_deleted" || table.Name == "_Field_deleted" {
						continue
					}
					actions = append(actions, "ALTER TABLE "+db.Quote(table.Name)+" DROP COLUMN "+db.Quote(oldFieldName))
				} else if !isPostgres {
					actions = append(actions, "DROP COLUMN "+db.Quote(oldFieldName))
				}
			}
		}

		if db.Config.Type == "mysql" {
			if len(pks) > 0 && strings.ToLower(oldIndexes["PRIMARY"]) != strings.ToLower(strings.Join(pks, " ")) {
				actions = append(actions, "ADD PRIMARY KEY(`"+strings.Join(pks, "`,`")+"`)")
			}
		}

		for keyId, keySet := range keySetBy {
			if oldIndexes[keyId] == "" || (!isPostgres && strings.ToLower(oldIndexes[keyId]) != strings.ToLower(keySetFields[keyId])) {
				if strings.HasPrefix(db.Config.Type, "sqlite") {
					// 针对 SQLite 优化：如果索引名不匹配，但存在一个相同列组的唯一索引（如 sqlite_autoindex），则跳过
					if strings.Contains(keySet, "UNIQUE") {
						foundUnique := false
						for oldName, oldCols := range oldIndexes {
							if oldUniqueMap[oldName] && strings.ToLower(oldCols) == strings.ToLower(keySetFields[keyId]) {
								foundUnique = true
								break
							}
						}
						if foundUnique {
							continue
						}
					}
					actions = append(actions, keySet)
				} else if isPostgres {
					actions = append(actions, keySet)
				} else {
					actions = append(actions, "ADD "+keySet)
				}
			}
		}

		if db.Config.Type == "mysql" {
			oldTableComment := cast.String(tableInfo["comment"])
			if table.Comment != oldTableComment {
				actions = append(actions, "COMMENT '"+table.Comment+"'")
			}
		}

		if len(actions) == 0 {
			goto SYNC_SHADOW
		}

		tx := db.schemaTx
		if tx == nil {
			tx = db.Begin()
			defer tx.CheckFinished()
		}
		var res *ExecResult
		if strings.HasPrefix(db.Config.Type, "sqlite") || isPostgres {
			for _, action := range actions {
				res = tx.Exec(action)
				if res.Error != nil {
					break
				}
			}
		} else {
			sql := "ALTER TABLE " + db.Quote(table.Name) + " " + strings.Join(actions, "\n,") + ";"
			res = tx.Exec(sql)
		}

		if res != nil && res.Error != nil {
			if db.schemaTx == nil {
				_ = tx.Rollback()
			}
			return res.Error
		}
		if db.schemaTx == nil {
			if err := tx.Commit(); err != nil {
				return err
			}
		}
	} else {
		// 创建新表
		if len(pks) > 0 {
			if isPostgres {
				// In Postgres, PK is often in CREATE TABLE
				fieldSets = append(fieldSets, "PRIMARY KEY ("+db.Quotes(pks)+")")
			} else {
				fieldSets = append(fieldSets, "PRIMARY KEY ("+db.Quotes(pks)+")")
			}
		}

		indexSets := make([]string, 0)
		if strings.HasPrefix(db.Config.Type, "sqlite") || isPostgres {
			for _, indexSql := range keySetBy {
				indexSets = append(indexSets, indexSql)
			}
		} else {
			for _, key := range keySets {
				fieldSets = append(fieldSets, key)
			}
		}

		sql := ""
		if strings.HasPrefix(db.Config.Type, "sqlite") || isPostgres {
			sql = fmt.Sprintf("CREATE TABLE %s (\n%s\n);", db.Quote(table.Name), strings.Join(fieldSets, ",\n"))
		} else if db.Config.Type == "mysql" {
			sql = fmt.Sprintf("CREATE TABLE `%s` (\n%s\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='%s';", table.Name, strings.Join(fieldSets, ",\n"), table.Comment)
		}

		tx := db.schemaTx
		if tx == nil {
			tx = db.Begin()
			defer tx.CheckFinished()
		}
		res := tx.Exec(sql)

		if res.Error == nil && (strings.HasPrefix(db.Config.Type, "sqlite") || isPostgres) {
			for _, indexSet := range indexSets {
				r := tx.Exec(indexSet)
				if r.Error != nil {
					res = r
					break
				}
			}
		}

		if res != nil && res.Error != nil {
			if db.schemaTx == nil {
				_ = tx.Rollback()
			}
			return res.Error
		}
		if db.schemaTx == nil {
			if err := tx.Commit(); err != nil {
				return err
			}
		}
	}

	if len(ftsFields) > 0 && strings.HasPrefix(db.Config.Type, "sqlite") {
		ftsTableName := table.Name + "_fts"
		ftsInfo := db.Query("SELECT \"name\" FROM \"sqlite_master\" WHERE \"type\"='table' AND \"name\"='" + ftsTableName + "'").MapOnR1()
		if ftsInfo["name"] == nil {
			// 创建 FTS 虚拟表
			if res := db.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE \"%s\" USING fts5(%s, tokenize='unicode61')", ftsTableName, strings.Join(ftsFields, ", "))); res.Error != nil {
				return res.Error
			}

			idField := "id"
			if len(pks) > 0 {
				idField = pks[0]
			}

			// AI Trigger
			newFtsFields := make([]string, 0, len(ftsFields))
			for _, f := range ftsFields {
				newFtsFields = append(newFtsFields, "new."+f)
			}
			aiSql := fmt.Sprintf("CREATE TRIGGER IF NOT EXISTS \"%s_ai\" AFTER INSERT ON \"%s\" BEGIN INSERT INTO \"%s\"(rowid, %s) VALUES (new.%s, %s); END;",
				ftsTableName, table.Name, ftsTableName, strings.Join(ftsFields, ", "), idField, strings.Join(newFtsFields, ", "))
			if res := db.Exec(aiSql); res.Error != nil {
				return res.Error
			}

			// AD Trigger
			adSql := fmt.Sprintf("CREATE TRIGGER IF NOT EXISTS \"%s_ad\" AFTER DELETE ON \"%s\" BEGIN DELETE FROM \"%s\" WHERE rowid = old.%s; END;",
				ftsTableName, table.Name, ftsTableName, idField)
			if res := db.Exec(adSql); res.Error != nil {
				return res.Error
			}

			// AU Trigger
			updateSets := make([]string, 0, len(ftsFields))
			for _, f := range ftsFields {
				updateSets = append(updateSets, fmt.Sprintf("%s = new.%s", f, f))
			}
			auSql := fmt.Sprintf("CREATE TRIGGER IF NOT EXISTS \"%s_au\" AFTER UPDATE ON \"%s\" BEGIN UPDATE \"%s\" SET %s WHERE rowid = old.%s; END;",
				ftsTableName, table.Name, ftsTableName, strings.Join(updateSets, ", "), idField)
			if res := db.Exec(auSql); res.Error != nil {
				return res.Error
			}
		}
	}
SYNC_SHADOW:
	if table.ShadowDelete && !strings.HasSuffix(table.Name, "_deleted") {
		table.HasShadowTable = true
		shadowTable := *table
		shadowTable.Name = table.Name + "_deleted"
		shadowTable.ShadowDelete = false
		shadowTable.Indexes = nil
		shadowTable.Fields = make([]TableField, 0, len(table.Fields)+1)
		for _, f := range table.Fields {
			f.Index = ""
			f.IndexGroup = ""
			f.Extra = ""
			if strings.Contains(f.Type, "serial") {
				if strings.Contains(f.Type, "bigserial") {
					f.Type = "bigint"
				} else {
					f.Type = "int"
				}
			}
			shadowTable.Fields = append(shadowTable.Fields, f)
		}
		// 自动为删除表增加删除时间字段，方便同步
		shadowTable.Fields = append(shadowTable.Fields, TableField{
			Name:    "deleted_at",
			Type:    "datetime",
			Default: "CURRENT_TIMESTAMP",
			Null:    "NOT NULL",
			Comment: "删除时间",
		})
		return db.CheckTable(&shadowTable)
	}
	return nil
}

type tableFieldDesc struct {
	Field   string
	Type    string
	Null    string
	Key     string
	Default string
	Extra   string
	After   string
}

type tableKeyDesc struct {
	Key_name    string
	Column_name string
}
