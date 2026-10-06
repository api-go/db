package db_test

import (
	"strings"
	"testing"

	"apigo.cc/go/db"
)

func TestSQLiteFTSStringPrimaryKeyAndLegacyTriggerMigration(t *testing.T) {
	store := db.GetDB("sqlite://"+t.TempDir()+"/string-fts.db", nil)
	const schema = "records\n  id c10 PK\n  body t TI\n"
	if err := store.Sync(schema); err != nil {
		t.Fatal(err)
	}
	check := func(query string, want int) {
		t.Helper()
		res := store.Query("SELECT id FROM records AS r WHERE r.body LIKE ?", "%"+query+"%")
		if res.Error != nil || len(res.MapResults()) != want {
			t.Fatalf("query %s want %d: %v", query, want, res.Error)
		}
	}
	if res := store.Insert("records", map[string]any{"id": "source0001", "body": "alpha migration"}); res.Error != nil {
		t.Fatal(res.Error)
	}
	check("alpha", 1)
	if res := store.Update("records", map[string]any{"body": "beta updated"}, "id=?", "source0001"); res.Error != nil {
		t.Fatal(res.Error)
	}
	check("alpha", 0)
	check("beta", 1)
	// Reproduce a deployed trigger that linked FTS rowid to a string primary key.
	for _, sql := range []string{
		"DROP TRIGGER records_fts_ai",
		"CREATE TRIGGER records_fts_ai AFTER INSERT ON records BEGIN INSERT INTO records_fts(rowid,body_tokens) VALUES(new.id,new.body_tokens); END",
		"DELETE FROM records_fts",
	} {
		if res := store.Exec(sql); res.Error != nil {
			t.Fatal(res.Error)
		}
	}
	if err := store.Sync(schema); err != nil {
		t.Fatal(err)
	}
	trigger := store.Query("SELECT sql FROM sqlite_master WHERE name='records_fts_ai'").StringOnR1C1()
	if !strings.Contains(trigger, "new.rowid") {
		t.Fatal("legacy trigger was not migrated")
	}
	check("beta", 1)
	if res := store.Insert("records", map[string]any{"id": "source0002", "body": "gamma inserted"}); res.Error != nil {
		t.Fatal(res.Error)
	}
	check("gamma", 1)
	if res := store.Delete("records", "id=?", "source0001"); res.Error != nil {
		t.Fatal(res.Error)
	}
	check("beta", 0)
	if err := store.Sync(schema); err != nil {
		t.Fatal(err)
	}
	check("gamma", 1)
	tx := store.Begin()
	if err := tx.Sync("records\n  id c10 PK\n"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	check("gamma", 1)
	if err := store.Sync("records\n  id c10 PK\n  body t\n  extra t TI\n"); err != nil {
		t.Fatal(err)
	}
	if res := store.Update("records", map[string]any{"extra": "delta newfield"}, "id=?", "source0002"); res.Error != nil {
		t.Fatal(res.Error)
	}
	res := store.Query("SELECT id FROM records WHERE extra LIKE ?", "%delta%")
	if res.Error != nil || len(res.MapResults()) != 1 {
		t.Fatalf("changed FTS fields did not work: %v", res.Error)
	}
	if err := store.Sync("records\n  id c10 PK\n"); err != nil {
		t.Fatal(err)
	}
	if count := store.Query("SELECT count(*) FROM sqlite_master WHERE name='records_fts' OR name LIKE 'records_fts_a%'").IntOnR1C1(); count != 0 {
		t.Fatal("removed FTS retained table/triggers")
	}
	if res := store.Insert("records", map[string]any{"id": "source0003"}); res.Error != nil {
		t.Fatal(res.Error)
	}
}
