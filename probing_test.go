package db

import (
	"testing"

	"github.com/api-go/file"
	_ "modernc.org/sqlite"
)

func TestTableProbing(t *testing.T) {
	ResetAllForTest()
	dbPath := "./test_probing.db"
	file.Remove(dbPath)
	SetConfigForTest("test_probing", &Config{Type: "sqlite", Host: dbPath})
	dbInst := GetDB("test_probing", nil)
	if dbInst == nil {
		t.Fatal("db is nil")
	}
	defer func() {
		dbInst.Destroy()
		file.Remove(dbPath)
	}()

	dbInst.Exec("CREATE TABLE users (id char(8) PRIMARY KEY, name TEXT, autoVersion BIGINT)")

	ts := dbInst.getTable("users")
	if ts.VersionField != "autoVersion" {
		t.Errorf("Expected version field 'autoVersion', got '%s'", ts.VersionField)
	}
	if ts.IdField != "id" {
		t.Errorf("Expected id field 'id', got '%s'", ts.IdField)
	}
	if ts.IdSize != 8 {
		t.Errorf("Expected id size 8, got %d", ts.IdSize)
	}
}
