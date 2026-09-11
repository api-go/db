package db_test

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"apigo.cc/go/cast"
	"apigo.cc/go/db"
	"apigo.cc/go/file"
	"apigo.cc/go/shell"

	_ "apigo.cc/go/db/mysql"
	_ "modernc.org/sqlite"
)

func TestMain(m *testing.M) {
	code := m.Run()
	file.Remove("test.db")
	os.Exit(code)
}

var dbset = "sqlite://test.db"

type userInfo struct {
	innerId int
	Tag     string
	Id      int
	Name    string
	Phone   *string
	Email   string
	Parents []string
	Active  bool
	Time    string
}

type UserBaseModel struct {
	Id       int
	Name     string
	Password string
}

type UserModel struct {
	UserBaseModel
	Phone      string
	Active     bool
	Parents    []string
	UserStatus int
	Owner      int
	Salt       string
}

func initDB(t *testing.T) *db.DB {
	var er *db.ExecResult
	dbInst := db.GetDB(dbset, nil)
	fmt.Println("dbType", shell.BCyan(dbInst.Config.Type))
	if dbInst.Error != nil {
		t.Fatal("GetDB error", dbInst)
		return nil
	}

	finishDB(dbInst, t)
	if dbInst.Config.Type == "mysql" {
		er = dbInst.Exec(`CREATE TABLE IF NOT EXISTS tempUsersForDBTest (
				id INT NOT NULL AUTO_INCREMENT,
				name VARCHAR(45) NOT NULL,
				phone VARCHAR(45),
				email VARCHAR(45),
				parents JSON,
				active TINYINT NOT NULL DEFAULT 0,
				time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				PRIMARY KEY (id));`)
	} else {
		er = dbInst.Exec(`CREATE TABLE IF NOT EXISTS tempUsersForDBTest (
				id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
				name VARCHAR(45) NOT NULL,
				phone VARCHAR(45),
				email VARCHAR(45),
				parents JSON,
				active TINYINT NOT NULL DEFAULT 0,
				time DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now', 'localtime')));`)
	}
	if er.Error != nil {
		t.Fatal("Failed to create table", er)
	}
	return dbInst
}

func finishDB(dbInst *db.DB, t *testing.T) {
	er := dbInst.Exec(`DROP TABLE IF EXISTS tempUsersForDBTest;`)
	if er.Error != nil {
		t.Fatal("Failed to drop table", er)
	}
}

func TestMakeInsertSql(t *testing.T) {
	user := &UserModel{
		UserBaseModel: UserBaseModel{
			Name:     "王二小",
			Password: "2121asds",
		},
		Parents:    []string{"aa", "bb"},
		Active:     true,
		UserStatus: 1,
		Salt:       "de312",
	}

	dbInst := db.GetDB(dbset, nil)
	query, _ := dbInst.MakeInsertSql("table_name", user, false)
	checkSql := `insert into "table_name" ("Id","Name","Password","Phone","Active","Parents","UserStatus","Owner","Salt") values (?,?,?,?,?,?,?,?,?)`
	if dbInst.Config.Type == "mysql" {
		checkSql = strings.ReplaceAll(checkSql, "\"", "`")
	}
	if query != checkSql {
		t.Fatal("MakeInsertSql query error ", query)
	}
}

func TestBaseSelect(t *testing.T) {
	sqlStr := "SELECT 1002 id, '13800000001' phone"
	dbInst := db.GetDB(dbset, nil)
	if dbInst.Error != nil {
		t.Fatal("GetDB error", dbInst.Error)
		return
	}

	r := dbInst.Query(sqlStr)
	if r.Error != nil {
		t.Fatal("Query error", sqlStr, r)
	}
	results1 := r.MapResults()
	if cast.Int(results1[0]["id"]) != 1002 || cast.String(results1[0]["phone"]) != "13800000001" {
		t.Fatal("Result error", sqlStr, results1, r)
	}

	r = dbInst.Query(sqlStr)
	if r.Error != nil {
		t.Fatal("Query error", sqlStr, r)
	}
	results2 := r.StringMapResults()
	if results2[0]["id"] != "1002" || results2[0]["phone"] != "13800000001" {
		t.Fatal("Result error", sqlStr, results2, r)
	}

	results3 := make([]map[string]int, 0)
	r = dbInst.Query(sqlStr)
	if r.Error != nil {
		t.Fatal("Query error", sqlStr, results3, r)
	}
	r.To(&results3)
	if results3[0]["id"] != 1002 || results3[0]["phone"] != 13800000001 {
		t.Fatal("Result error", sqlStr, results3, r)
	}

	results4 := make([]userInfo, 0)
	r = dbInst.Query(sqlStr)
	if r.Error != nil {
		t.Fatal("Query error", sqlStr, results4, r)
	}
	r.To(&results4)
	if results4[0].Id != 1002 || results4[0].Phone == nil || *results4[0].Phone != "13800000001" {
		t.Fatal("Result error", sqlStr, results4, r)
	}

	results5 := dbInst.Query(sqlStr).StringSliceResults()
	if results5[0][0] != "1002" || results5[0][1] != "13800000001" {
		t.Fatal("Result error", sqlStr, results5, r)
	}

	r = dbInst.Query(sqlStr)
	if r.Error != nil {
		t.Fatal("Query error", sqlStr, r)
	}
	results6 := r.StringsOnC1()
	if results6[0] != "1002" {
		t.Fatal("Result error", sqlStr, results6, r)
	}

	r = dbInst.Query(sqlStr)
	if r.Error != nil {
		t.Fatal("Query error", sqlStr, r)
	}
	results7 := r.MapOnR1()
	if cast.Int(results7["id"]) != 1002 || cast.String(results7["phone"]) != "13800000001" {
		t.Fatal("Result error", sqlStr, results7, r)
	}

	results8 := userInfo{innerId: 2, Tag: "abc"}
	r = dbInst.Query(sqlStr)
	if r.Error != nil {
		t.Fatal("Query error", sqlStr, results8, r)
	}
	r.To(&results8)
	if results8.Id != 1002 || results8.Phone == nil || *results8.Phone != "13800000001" || results8.innerId != 2 || results8.Tag != "abc" {
		t.Fatal("Result error", sqlStr, results8, r)
	}

	r = dbInst.Query(sqlStr)
	if r.Error != nil {
		t.Fatal("Query error", sqlStr, r)
	}
	results9 := r.IntOnR1C1()
	if results9 != 1002 {
		t.Fatal("Result error", sqlStr, results9, r)
	}

	r = dbInst.Query(sqlStr)
	results10 := map[string]string{}
	r.ToKV(&results10)
	if results10["1002"] != "13800000001" {
		t.Fatal("Result error", sqlStr, results10, r)
	}

	r = dbInst.Query(sqlStr)
	results11 := map[string]map[string]string{}
	r.ToKV(&results11)
	if results11["1002"]["phone"] != "13800000001" {
		t.Fatal("Result error", sqlStr, results11, r)
	}

	r = dbInst.Query(sqlStr)
	results12 := map[string]userInfo{}
	r.ToKV(&results12)
	if results12["1002"].Phone == nil || *results12["1002"].Phone != "13800000001" {
		t.Fatal("Result error", sqlStr, results12, r)
	}
}

func TestInsertReplaceUpdateDelete(t *testing.T) {
	dbInst := initDB(t)
	data := map[string]any{
		"phone":   18033336666,
		"name":    "Star",
		"parents": []string{"dd", "mm"},
		"time":    ":(strftime('%Y-%m-%d %H:%M:%f', datetime('now', '-1 day'), 'localtime'))",
	}
	if dbInst.Config.Type == "mysql" {
		data["time"] = ":DATE_SUB(NOW(), INTERVAL 1 DAY)"
	}
	er := dbInst.Insert("tempUsersForDBTest", data)
	if er.Error != nil {
		t.Fatal("Insert 1 error", er)
	}
	if er.Id() != 1 {
		t.Fatal("insertId 1 error", er, er.Id())
	}

	er = dbInst.Insert("tempUsersForDBTest", map[string]any{
		"phone":  "18000000002",
		"name":   "Tom",
		"active": true,
	})
	if er.Error != nil {
		t.Fatal("Insert 2 error", er)
	}
	if er.Id() != 2 {
		t.Fatal("insertId 2 error", er, er.Id())
	}

	er = dbInst.Update("tempUsersForDBTest", map[string]any{
		"phone": "18000000222",
		"name":  "Tom Lee",
	}, "id=?", 2)
	if er.Error != nil {
		t.Fatal("Update 2 error", er)
	}
	if er.Changes() != 1 {
		t.Fatal("Update 2 num error", er, er.Changes())
	}

	er = dbInst.Replace("tempUsersForDBTest", map[string]any{
		"phone": "18000000003",
		"name":  "Amy",
	})
	if er.Error != nil {
		t.Fatal("Replace 3 error", er)
	}
	if er.Id() != 3 {
		t.Fatal("insertId 3 error", er, er.Changes())
	}

	er = dbInst.Exec("delete from tempUsersForDBTest where id=3")
	if er.Error != nil {
		t.Fatal("Delete 3 error", er)
	}
	if er.Changes() != 1 {
		t.Fatal("Delete 3 num error", er)
	}

	er = dbInst.Replace("tempUsersForDBTest", map[string]any{
		"phone": "18000000004",
		"name":  "Jerry",
	})
	if er.Error != nil {
		t.Fatal("Replace 4 error", er)
	}
	if er.Id() != 4 {
		t.Fatal("insertId 4 error", er, er.Changes())
	}

	stmt := dbInst.Prepare("replace into `tempUsersForDBTest` (`id`,`phone`,`name`) values (?,?,?)")
	if stmt.Error != nil {
		t.Fatal("Prepare 4 error", stmt)
	}
	er = stmt.Exec(4, "18000000004", "Jerry's Mather")
	stmt.Close()

	if er.Error != nil {
		t.Fatal("Replace 4 error", er)
	}
	if er.Id() != 4 {
		t.Fatal("insertId 4 error", er)
	}

	userList := make([]userInfo, 0)
	r := dbInst.Query("select * from tempUsersForDBTest")
	if r.Error != nil {
		t.Fatal("Select userList error", r)
	}
	r.To(&userList)
	fmt.Println(">>>>", cast.PrettyToJSON(userList))
	if strings.Split(userList[0].Time, " ")[0] != time.Now().Add(time.Hour*24*-1).Format("2006-01-02") || userList[0].Id != 1 || userList[0].Name != "Star" || userList[0].Phone == nil || *userList[0].Phone != "18033336666" || userList[0].Active != false {
		t.Fatal("Select userList 0 error", userList, r)
	}
	if len(userList[0].Parents) != 2 || userList[0].Parents[0] != "dd" {
		t.Fatal("Select userList 0 Parents error", userList, r)
	}
	if strings.Split(userList[1].Time, " ")[0] != time.Now().Format("2006-01-02") || userList[1].Id != 2 || userList[1].Name != "Tom Lee" || userList[1].Phone == nil || *userList[1].Phone != "18000000222" || userList[1].Active != true {
		t.Fatal("Select userList 1 error", userList, r)
	}
	if userList[2].Id != 4 || userList[2].Name != "Jerry's Mather" || userList[2].Phone == nil || *userList[2].Phone != "18000000004" {
		t.Fatal("Select userList 2 error", userList, r)
	}

	finishDB(dbInst, t)
}

func TestMakeKeysVarsValuesBindsIPv6Loopback(t *testing.T) {
	_, vars, values := db.MakeKeysVarsValues(map[string]any{"clientIP": "::1"})
	if len(vars) != 1 || vars[0] != "?" {
		t.Fatalf("IPv6 loopback must use a bound parameter, got vars %#v", vars)
	}
	if len(values) != 1 || values[0] != "::1" {
		t.Fatalf("IPv6 loopback value = %#v", values)
	}
}

func TestTransaction(t *testing.T) {
	n1 := countConnection()

	var userList []userInfo

	dbInst := initDB(t)
	tx := dbInst.Begin()
	if tx.Error != nil {
		t.Fatal("Begin error", tx)
	}

	data := map[string]any{
		"phone": 18033336666,
		"name":  "Star",
		"time":  ":(strftime('%Y-%m-%d %H:%M:%f', datetime('now', '-1 day'), 'localtime'))",
	}
	if dbInst.Config.Type == "mysql" {
		data["time"] = ":DATE_SUB(NOW(), INTERVAL 1 DAY)"
	}
	tx.Insert("tempUsersForDBTest", data)

	userList = make([]userInfo, 0)
	r := dbInst.Query("select * from tempUsersForDBTest")
	r.To(&userList)
	if r.Error != nil || len(userList) != 0 {
		t.Fatal("Select Out Of TX", userList, r)
	}

	userList = make([]userInfo, 0)
	r = tx.Query("select * from tempUsersForDBTest")
	r.To(&userList)
	if r.Error != nil || len(userList) != 1 {
		t.Fatal("Select In TX", userList, r)
	}

	tx.Rollback()

	userList = make([]userInfo, 0)
	r = dbInst.Query("select * from tempUsersForDBTest")
	r.To(&userList)
	if r.Error != nil || len(userList) != 0 {
		t.Fatal("Select When Rollback", userList, r)
	}

	tx = dbInst.Begin()
	defer func() {
		if err := tx.Finish(false); err != nil {
			t.Error("tx rollback error", err)
		}
		finishDB(dbInst, t)
	}()
	if tx.Error != nil {
		t.Fatal("Begin 2 error", tx)
	}

	stmt := tx.Prepare("insert into `tempUsersForDBTest` (`id`,`phone`,`name`) values (?,?,?)")
	if stmt.Error != nil {
		t.Fatal("Prepare 4 error", r)
	}
	stmt.Exec(4, "18000000004", "Jerry's Mather")
	stmt.Close()

	tx.Commit()

	userList = make([]userInfo, 0)
	r = dbInst.Query("select * from tempUsersForDBTest")
	r.To(&userList)
	if r.Error != nil || len(userList) != 1 {
		t.Fatal("Select When Commit", userList, r)
	}

	n2 := countConnection()
	fmt.Println("# connection count", n1, n2, cast.As(cast.ToJSON(dbInst.GetOriginDB().Stats())), ".")
}

func countConnection() int {
	n := 0
	res, _ := shell.RunCommand("netstat -ant", nil)
	lines := strings.Split(string(res.Stdout), "\n")
	spliter := regexp.MustCompile("\\s+")
	for _, line := range lines {
		if strings.Contains(line, ".3306") && strings.Contains(line, "ESTABLISHED") {
			a := spliter.Split(line, 10)
			if len(a) > 4 && strings.Contains(a[4], ".3306") {
				n++
			}
		}
	}
	return n
}

func BenchmarkForPool(b *testing.B) {
	b.StopTimer()
	sqlStr := "SELECT 1002 id, '13800000001' phone"
	dbInst := db.GetDB(dbset, nil)
	if dbInst.Error != nil {
		b.Fatal("GetDB error", dbInst)
		return
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		results1 := make([]map[string]any, 0)
		r := dbInst.Query(sqlStr)
		if r.Error != nil {
			b.Fatal("Query error", sqlStr, results1, r)
		}
		r.To(&results1)
		if cast.Int(results1[0]["id"]) != 1002 || cast.String(results1[0]["phone"]) != "13800000001" {
			b.Fatal("Result error", sqlStr, results1, r)
		}
	}
	b.Log("OpenConnections", dbInst.GetOriginDB().Stats().OpenConnections)
}

func BenchmarkForPoolParallel(b *testing.B) {
	n1 := countConnection()

	b.StopTimer()
	sqlStr := "SELECT 1002 id, '13800000001' phone"
	dbInst := db.GetDB(dbset, nil)
	if dbInst.Error != nil {
		b.Fatal("GetDB error", dbInst)
		return
	}
	b.StartTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			results1 := make([]map[string]any, 0)
			r := dbInst.Query(sqlStr)
			if r.Error != nil {
				b.Fatal("Query error", sqlStr, results1, r)
			}
			r.To(&results1)
			if cast.Int(results1[0]["id"]) != 1002 || cast.String(results1[0]["phone"]) != "13800000001" {
				b.Fatal("Result error", sqlStr, results1, r)
			}
		}
	})
	b.Log("OpenConnections", dbInst.GetOriginDB().Stats().OpenConnections)

	n2 := countConnection()
	fmt.Println("# connection count", n1, n2, cast.As(cast.ToJSON(dbInst.GetOriginDB().Stats())), ".")
}
