package db_test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"apigo.cc/go/db"
	_ "apigo.cc/go/db/pgx"
)

func TestPostgresSchemaIndexTransitions(t *testing.T) {
	dsn := os.Getenv("DB_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set DB_TEST_POSTGRES_DSN for isolated real PostgreSQL index lifecycle")
	}
	store := db.GetDB(dsn, nil)
	name := fmt.Sprintf("db_index_transition_%d", time.Now().UnixNano())
	t.Cleanup(func() {
		if err := store.Exec("DROP TABLE IF EXISTS " + store.Quote(name)).Error; err != nil {
			t.Error(err)
		}
	})
	sync := func(index string) error {
		return store.Sync(name + "\n  id i PK\n  \"备件名称\" v100 " + index + "\n")
	}
	check := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	indexes := func() map[string]any {
		result := store.Query("SELECT indexname,indexdef FROM pg_indexes WHERE schemaname='public' AND tablename=?", name)
		out := map[string]any{}
		check(result.ToKV(&out))
		return out
	}
	check(sync("U"))
	check(store.Insert(name, map[string]any{"id": 1, "备件名称": "0012"}).Error)
	if err := store.Insert(name, map[string]any{"id": 2, "备件名称": "0012"}).Error; err == nil {
		t.Fatal("unique must reject duplicate")
	}
	// External indexes (even with a managed-looking name) and constraint-backed
	// indexes are outside the schema's ownership.
	check(store.Exec("CREATE INDEX " + store.Quote("external_"+name) + " ON " + store.Quote(name) + " (id)").Error)
	check(store.Exec("CREATE INDEX " + store.Quote("ik_"+name+"_external") + " ON " + store.Quote(name) + " (id)").Error)
	check(store.Exec("CREATE INDEX " + store.Quote("ik_"+name+"_id") + " ON " + store.Quote(name) + " (id) WHERE id > 0").Error)
	check(store.Exec("ALTER TABLE " + store.Quote(name) + " ADD CONSTRAINT " + store.Quote("uk_"+name+"_external") + " UNIQUE (id)").Error)
	check(sync("I"))
	if indexes()["uk_"+name+"_备件名称"] != nil {
		t.Fatal("obsolete unique index remains after Unique→Index")
	}
	check(store.Insert(name, map[string]any{"id": 2, "备件名称": "0012"}).Error)
	if err := sync("U"); err == nil {
		t.Fatal("Index→Unique must fail with duplicates")
	}
	if indexes()["ik_"+name+"_备件名称"] == nil {
		t.Fatal("failed migration did not roll back old index")
	}
	check(sync("I")) // idempotent after rollback
	check(sync(""))
	got := indexes()
	if got["ik_"+name+"_备件名称"] != nil {
		t.Fatal("obsolete ordinary index remains after Index→None")
	}
	for _, preserved := range []string{name + "_pkey", "external_" + name, "ik_" + name + "_external", "ik_" + name + "_id", "uk_" + name + "_external"} {
		if got[preserved] == nil {
			t.Fatalf("unowned/constraint index %s removed", preserved)
		}
	}
	check(store.Delete(name, "id=?", 2).Error)
	check(sync("U"))
	if err := store.Insert(name, map[string]any{"id": 3, "备件名称": "0012"}).Error; err == nil {
		t.Fatal("recreated unique must enforce constraint")
	}
	if count := store.Query("SELECT COUNT(*) FROM " + store.Quote(name)).IntOnR1C1(); count != 1 {
		t.Fatalf("unexpected retained rows: %d", count)
	}
}

func TestPostgresLongSchemaIndexTransitions(t *testing.T) {
	dsn := os.Getenv("DB_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set DB_TEST_POSTGRES_DSN for isolated real PostgreSQL index lifecycle")
	}
	store := db.GetDB(dsn, nil)
	name := strings.Repeat("l", 36) + fmt.Sprint(time.Now().UnixNano()) // 55 bytes; managed names are hashed.
	t.Cleanup(func() {
		if err := store.Exec("DROP TABLE IF EXISTS " + store.Quote(name)).Error; err != nil {
			t.Error(err)
		}
	})
	sync := func(index string) {
		t.Helper()
		if err := store.Sync(name + "\n  id i PK\n  value v100 " + index + "\n"); err != nil {
			t.Fatal(err)
		}
	}
	sync("U")
	if err := store.Insert(name, map[string]any{"id": 1, "value": "0012"}).Error; err != nil {
		t.Fatal(err)
	}
	sync("I")
	if err := store.Insert(name, map[string]any{"id": 2, "value": "0012"}).Error; err != nil {
		t.Fatal("hashed unique index remains:", err)
	}
	sync("")
	result := store.Query("SELECT COUNT(*) FROM pg_indexes WHERE schemaname='public' AND tablename=?", name)
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	if count := result.IntOnR1C1(); count != 1 {
		t.Fatalf("expected only primary key index after removing hashed index; got %d", count)
	}
}
