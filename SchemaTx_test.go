package db_test

import (
	"path/filepath"
	"testing"

	"apigo.cc/go/db"
)

func TestSchemaTransactionRollbackAndCommit(t *testing.T) {
	d := db.GetDB("sqlite://"+filepath.Join(t.TempDir(), "schema-tx.db"), nil)
	defer d.Destroy()
	base := "records SD\n  id i PK\n  code v32\n"
	if err := d.Sync(base); err != nil {
		t.Fatal(err)
	}
	if r := d.Exec("INSERT INTO records(id,code) VALUES(1,'same'),(2,'same')"); r.Error != nil {
		t.Fatal(r.Error)
	}
	tx := d.Begin()
	if r := tx.Exec("UPDATE records SET code='changed' WHERE id=1"); r.Error != nil {
		t.Fatal(r.Error)
	}
	if err := tx.Sync(base + "  added v32\n  code U\n"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if d.Query("SELECT code FROM records WHERE id=1").StringOnR1C1() != "same" {
		t.Fatal("row did not roll back")
	}
	for _, row := range d.Query("PRAGMA table_info(records)").MapResults() {
		if row["name"] == "added" {
			t.Fatal("column did not roll back")
		}
	}
	tx = d.Begin()
	if err := tx.Sync(base + "  added v32\n  code U\n"); err == nil {
		t.Fatal("duplicate unique index accepted")
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if d.Query("SELECT count(*) FROM pragma_index_list('records') WHERE origin='c'").IntOnR1C1() != 0 {
		t.Fatal("failed index leaked")
	}
	tx = d.Begin()
	if err := tx.Sync(base + "  added v32\n  code I\n"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if d.Query("SELECT count(*) FROM pragma_index_list('records') WHERE origin='c'").IntOnR1C1() != 1 {
		t.Fatal("committed index missing")
	}
}
