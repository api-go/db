package db_test

import (
	"testing"

	"github.com/api-go/db"
	"github.com/api-go/file"
	_ "modernc.org/sqlite"
)

func TestParseFieldQuotedName(t *testing.T) {
	field := db.ParseField(`"Display Name 中文" v255 I // label`)
	if field.Name != "Display Name 中文" || field.Type != "varchar(255)" || field.Index != "index" || field.Comment != "label" {
		t.Fatalf("unexpected quoted field: %#v", field)
	}
}

func TestNumericIndexGroupsAreRejected(t *testing.T) {
	store := db.GetDB("sqlite://"+t.TempDir()+"/numeric-group.db", nil)
	if err := store.Sync("records\n  id c10 PK\n  content t TI7\n"); err == nil {
		t.Fatal("numeric index group must be rejected")
	}
}

func TestSchemaSync(t *testing.T) {
	dbPath := "test_schema.db"
	dbInst := db.GetDB("sqlite://"+dbPath, nil)
	defer file.Remove(dbPath)
	defer dbInst.Exec("DROP TABLE IF EXISTS test_table")
	defer dbInst.Exec("DROP TABLE IF EXISTS test_table_deleted")

	schema := `== Default ==
test_table SD // Test table with shadow delete
  id i AI // ID
  name v50 U
  autoVersion ubi // Version
  status ti // Status
`
	err := dbInst.Sync(schema)
	if err != nil {
		t.Fatal("Sync error:", err)
	}

	dbInst.Insert("test_table", map[string]any{"name": "test", "status": 1})
	dbInst.Delete("test_table", "id=?", 1)
	res := dbInst.Query("SELECT COUNT(*) FROM test_table_deleted")
	if res.IntOnR1C1() != 1 {
		t.Fatal("Shadow delete failed")
	}

	withoutStatus := `== Default ==
test_table SD // Test table with shadow delete
  id i AI // ID
  name v50 U
  autoVersion ubi // Version
`
	if err := dbInst.Sync(withoutStatus); err != nil {
		t.Fatal("Sync removed field:", err)
	}
	for _, tableName := range []string{"test_table", "test_table_deleted"} {
		fields := dbInst.Query("PRAGMA table_info(" + dbInst.Quote(tableName) + ")").MapResults()
		for _, field := range fields {
			if field["name"] == "status" {
				t.Fatalf("removed field remains in physical table %s", tableName)
			}
		}
	}
}

func TestAutoDetectShadow(t *testing.T) {
	dbPath := "auto_detect.db"
	dbInst := db.GetDB("sqlite://"+dbPath, nil)
	defer file.Remove(dbPath)
	defer dbInst.Exec("DROP TABLE IF EXISTS test_auto")
	defer dbInst.Exec("DROP TABLE IF EXISTS test_auto_deleted")

	dbInst.Exec("CREATE TABLE test_auto (id INTEGER PRIMARY KEY)")
	dbInst.Exec("CREATE TABLE test_auto_deleted (id INTEGER PRIMARY KEY)")
	dbInst.Insert("test_auto", map[string]any{"id": 1})
	dbInst.Delete("test_auto", "id=?", 1)
	res := dbInst.Query("SELECT COUNT(*) FROM test_auto_deleted")
	if res.IntOnR1C1() != 1 {
		t.Fatal("Auto-detect shadow delete failed")
	}
}
