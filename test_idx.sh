rm -f test_idx.db
sqlite3 test_idx.db "CREATE TABLE test (name TEXT);"
sqlite3 test_idx.db "CREATE INDEX idx_name ON test(name);"
sqlite3 test_idx.db "CREATE UNIQUE INDEX IF NOT EXISTS idx_name ON test(name);"
echo "Result code: $?"
