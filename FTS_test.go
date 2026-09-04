package db_test

import (
	"strings"
	"testing"

	"apigo.cc/go/db"
	"apigo.cc/go/file"
	_ "modernc.org/sqlite"
)

func TestAutonomousFTS(t *testing.T) {
	dbPath := "test_fts.db"
	dbInst := db.GetDB("sqlite://"+dbPath, nil)
	defer file.Remove(dbPath)
	defer dbInst.Exec("DROP TABLE IF EXISTS fts_test")
	defer dbInst.Exec("DROP TABLE IF EXISTS fts_test_fts")

	schema := `== Default ==
fts_test
  id i AI
  title t // Fulltext title
  content t // Fulltext content
  status i
  title TI
  content TI
`
	err := dbInst.Sync(schema)
	if err != nil {
		t.Fatal("Sync error:", err)
	}

	// 1. Verify schema
	row := dbInst.Query("SELECT \"sql\" FROM \"sqlite_master\" WHERE \"type\"='table' AND \"name\"='fts_test'").MapOnR1()
	sqlStr := ""
	if row["sql"] != nil {
		sqlStr = row["sql"].(string)
	}
	if !strings.Contains(sqlStr, "title_tokens") || !strings.Contains(sqlStr, "content_tokens") {
		t.Fatalf("Shadow columns missing in main table: %s", sqlStr)
	}

	row = dbInst.Query("SELECT \"name\" FROM \"sqlite_master\" WHERE \"type\"='table' AND \"name\"='fts_test_fts'").MapOnR1()
	if row["name"] == nil {
		t.Fatal("FTS virtual table missing")
	}

	// 2. Test Insert
	dbInst.Insert("fts_test", map[string]any{
		"title":   "你好世界",
		"content": "这是一段测试文本",
		"status":  1,
	})

	// Check if tokens are populated in main table
	row = dbInst.Query("SELECT title_tokens, content_tokens FROM fts_test WHERE id=1").MapOnR1()
	if row["title_tokens"] == nil || row["title_tokens"] == "" {
		t.Fatal("Tokens not populated in main table")
	}

	// Check if tokens are in FTS table
	row = dbInst.Query("SELECT * FROM fts_test_fts").MapOnR1()
	if row["title_tokens"] == nil || row["title_tokens"] == "" {
		t.Fatal("Tokens not populated in FTS table")
	}

	// 3. Test Query Interception (LIKE -> FTS)
	// Searching for "世界" should match "你好世界"
	res := dbInst.Query("SELECT * FROM fts_test WHERE title LIKE ?", "%世界%")
	list := res.MapResults()
	if len(list) != 1 {
		t.Fatalf("Query failed to find match via FTS redirection, found %d", len(list))
	}

	// 4. Test Update
	dbInst.Update("fts_test", map[string]any{"title": "更新后的标题"}, "id=?", 1)
	row = dbInst.Query("SELECT title_tokens FROM fts_test WHERE id=1").MapOnR1()
	if !strings.Contains(row["title_tokens"].(string), "更新") {
		t.Fatalf("Tokens not updated: %v", row["title_tokens"])
	}

	// 5. Test Multiple Fields & Alias
	dbInst.Insert("fts_test", map[string]any{
		"title":   "测试标题",
		"content": "北京大学是一个好学校",
		"status":  1,
	})

	// Search in content using alias
	res = dbInst.Query("SELECT t.title FROM fts_test AS t WHERE t.content LIKE ?", "%北京大学%")
	list = res.MapResults()
	if len(list) != 1 {
		t.Fatalf("Alias query failed, found %d", len(list))
	}

	// 6. Test Hardcoded Literals
	res = dbInst.Query("SELECT * FROM fts_test WHERE title LIKE '%标题%'")
	list = res.MapResults()
	if len(list) != 2 {
		t.Fatalf("Hardcoded literal query failed, found %d", len(list))
	}

	// 7. Test Various Identifier Styles
	styles := []string{
		"SELECT * FROM fts_test WHERE `title` LIKE ?",
		"SELECT * FROM fts_test WHERE \"title\" LIKE ?",
		"SELECT * FROM fts_test WHERE 'title' LIKE ?",
		"SELECT * FROM fts_test WHERE `fts_test`.`title` LIKE ?",
	}
	for _, sql := range styles {
		res = dbInst.Query(sql, "%测试%")
		list = res.MapResults()
		if len(list) != 1 {
			t.Errorf("Style failed: %s, found %d", sql, len(list))
		}
	}
}
