package db

import (
	"strings"
	"testing"
)

func TestPostgresFieldTypes(t *testing.T) {
	tests := map[string]string{
		"enabled b":  `"enabled" boolean`,
		"count ui":   `"count" integer`,
		"total ubi":  `"total" bigint`,
		"score uff":  `"score" double precision`,
		"created dt": `"created" timestamp`,
		"payload bb": `"payload" bytea`,
		"value o":    `"value" jsonb`,
	}
	for schema, expected := range tests {
		field := ParseField(schema)
		field.Parse("postgres")
		if !strings.Contains(field.Desc, expected) {
			t.Errorf("Parse(%q).Desc = %q, want it to contain %q", schema, field.Desc, expected)
		}
	}
}

func TestRewritePostgresQuery(t *testing.T) {
	query := "SELECT `id`, createTime FROM `items` WHERE tableId=? AND `memo`='literal ? and `tick`' AND `id`=?"
	want := `SELECT "id", "createTime" FROM "items" WHERE "tableId"=$1 AND "memo"='literal ? and ` + "`tick`" + `' AND "id"=$2`
	if got := rewritePostgresQuery(query); got != want {
		t.Fatalf("rewrite = %q, want %q", got, want)
	}
}

func TestCanonicalColumnNames(t *testing.T) {
	keys, _, values := canonicalColumnValues(
		[]string{"Name", "enabled", "Enabled", "name", "runas", "unknown"},
		[]string{"?", "?", "?", "?", "?", "?"},
		[]any{"old", true, false, "new", "runner", "value"},
		&TableStruct{Columns: []string{"Name", "Enabled", "RunAs"}},
	)
	want := []string{"Name", "Enabled", "RunAs", "unknown"}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("keys = %#v, want %#v", keys, want)
		}
	}
	if values[0] != "new" || values[1] != true {
		t.Fatalf("values = %#v, want alias values to override canonical values", values)
	}
}

func TestMySQLLongStringIndexUsesPrefix(t *testing.T) {
	table := &TableStruct{Fields: []TableField{{Name: "name", Type: "varchar(1024)"}}}
	if got := (&DB{QuoteTag: "`"}).mysqlIndexColumns(table, []string{"name"}); got != "`name`(768)" {
		t.Fatalf("mysql index columns = %q", got)
	}
}
