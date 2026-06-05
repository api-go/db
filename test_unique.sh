rm -f test_unique.db
sqlite3 test_unique.db "CREATE TABLE test (name TEXT);"
sqlite3 test_unique.db "INSERT INTO test (name) VALUES ('a');"
sqlite3 test_unique.db "INSERT INTO test (name) VALUES ('b');"
sqlite3 test_unique.db "CREATE UNIQUE INDEX uk_test_name ON test(name);"
echo "Result code: $?"
