package db

import (
	"fmt"
	"regexp"
	"strings"
)

var sizedSchemaType = regexp.MustCompile(`^[cv][1-9][0-9]*$`)
var nativeSizedSchemaType = regexp.MustCompile(`^(varchar|char)\(([1-9][0-9]*)\)$`)

// TableDB reconstructs schemas from parsed metadata, which stores native
// type names. Accept this finite spelling set, not arbitrary SQL fragments.
func canonicalFieldParts(parts []string) []string {
	if len(parts) < 2 {
		return parts
	}
	parts = append([]string{}, parts...)
	typeName := parts[1]
	if len(parts) > 2 && parts[2] == "unsigned" {
		typeName += " unsigned"
	}
	types := map[string]string{"char": "c", "varchar": "v", "datetime": "dt", "date": "d", "time": "tm", "int": "i", "integer": "i", "int unsigned": "ui", "tinyint": "ti", "tinyint unsigned": "uti", "bigint": "bi", "bigint unsigned": "ubi", "float": "f", "float unsigned": "uf", "double": "ff", "double unsigned": "uff", "smallint": "si", "smallint unsigned": "usi", "middleint": "mi", "middleint unsigned": "umi", "text": "t", "json": "o", "blob": "bb"}
	if token := types[typeName]; token != "" {
		parts[1] = token
		if strings.HasSuffix(typeName, " unsigned") {
			parts = append(parts[:2], parts[3:]...)
		}
	} else if matched := nativeSizedSchemaType.FindStringSubmatch(typeName); matched != nil {
		parts[1] = matched[1][:1] + matched[2]
	}
	return parts
}

func schemaType(token string) bool {
	switch token {
	case "c", "v", "dt", "d", "tm", "i", "ui", "ti", "uti", "b", "bi", "ubi", "f", "uf", "ff", "uff", "si", "usi", "mi", "umi", "t", "o", "bb", "ver":
		return true
	}
	return sizedSchemaType.MatchString(token)
}

func validateSchemaField(parts []string) error {
	if len(parts) < 2 || !schemaType(parts[1]) {
		return fmt.Errorf("field definition requires a supported type: %s", strings.Join(parts, " "))
	}
	index, nullable, timestamp := false, false, false
	for _, token := range parts[2:] {
		switch token {
		case "PK", "AI", "I", "U", "TI":
			if index {
				return fmt.Errorf("multiple field index markers: %s", parts[0])
			}
			index = true
		case "n", "nn":
			if nullable {
				return fmt.Errorf("multiple nullability markers: %s", parts[0])
			}
			nullable = true
		case "ct", "ctu":
			if timestamp {
				return fmt.Errorf("multiple timestamp defaults: %s", parts[0])
			}
			timestamp = true
		default:
			// Removed with the TableDB caller migration; do not break the
			// running caller halfway through the staged index conversion.
			legacy := wnMatcher.FindStringSubmatch(token)
			if legacy == nil || (legacy[1] != "I" && legacy[1] != "U" && legacy[1] != "TI") || index {
				return fmt.Errorf("unsupported field marker %q on %s", token, parts[0])
			}
			index = true
		}
	}
	return nil
}

// A declaration has no field type: "a,b I" or "-a I". Single ascending
// indexes can also be written as "a I" without redefining the field.
func parseTableIndex(line string) (TableIndex, bool, error) {
	line = strings.TrimSpace(strings.SplitN(line, "//", 2)[0])
	words := strings.Fields(line)
	if len(words) == 0 {
		return TableIndex{}, false, nil
	}
	// An exact type token makes this a field declaration, regardless of
	// its trailing index marker. Otherwise only index grammar is allowed.
	fieldParts := canonicalFieldParts(splitSchemaField(line))
	if len(fieldParts) > 1 && schemaType(fieldParts[1]) {
		return TableIndex{}, false, nil
	}
	kind := words[len(words)-1]
	isKind := kind == "I" || kind == "U" || kind == "TI"
	declaration := strings.Contains(line, ",") || strings.HasPrefix(words[0], "-") || isKind
	if !declaration {
		return TableIndex{}, false, nil
	}
	if !isKind || len(words) < 2 {
		return TableIndex{}, true, fmt.Errorf("invalid index declaration: %s", line)
	}
	columns := strings.TrimSpace(strings.TrimSuffix(line, kind))
	fields := strings.Split(columns, ",")
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	return TableIndex{Type: kind, Fields: fields}, true, nil
}

// Validate before schema synchronization can change any physical columns.
func (table *TableStruct) prepareIndexes() error {
	if table.indexError != nil {
		return table.indexError
	}
	known := make(map[string]bool, len(table.Fields))
	for _, field := range table.Fields {
		if field.parseError != nil {
			return field.parseError
		}
		if known[field.Name] {
			return fmt.Errorf("table %s: duplicate field %q", table.Name, field.Name)
		}
		known[field.Name] = true
	}
	for _, index := range table.Indexes {
		kind := strings.ToUpper(index.Type)
		if kind != "I" && kind != "U" && kind != "TI" {
			return fmt.Errorf("table %s: invalid index type %q", table.Name, index.Type)
		}
		if len(index.Fields) == 0 {
			return fmt.Errorf("table %s: empty index", table.Name)
		}
		seen := map[string]bool{}
		for _, column := range index.Fields {
			name := strings.TrimPrefix(column, "-")
			if name == "" || !known[name] {
				return fmt.Errorf("table %s: index references unknown field %q", table.Name, name)
			}
			if seen[name] {
				return fmt.Errorf("table %s: repeated index field %q", table.Name, name)
			}
			seen[name] = true
			if kind == "TI" && strings.HasPrefix(column, "-") {
				return fmt.Errorf("table %s: fulltext indexes cannot specify descending order", table.Name)
			}
		}
	}
	// Fulltext declarations use the same token-column boundary as field TI.
	for _, index := range table.Indexes {
		if strings.ToUpper(index.Type) != "TI" {
			continue
		}
		for _, name := range index.Fields {
			token := name + "_tokens"
			if !known[token] {
				table.Fields = append(table.Fields, TableField{Name: token, Type: "text", Null: "NULL", Comment: "FTS tokens for " + name})
				known[token] = true
			}
		}
	}
	return nil
}

func (db *DB) indexColumns(fields []string) string {
	quoted := make([]string, len(fields))
	for i, field := range fields {
		quoted[i] = db.Quote(strings.TrimPrefix(field, "-"))
		if strings.HasPrefix(field, "-") {
			quoted[i] += " DESC"
		}
	}
	return strings.Join(quoted, ", ")
}
