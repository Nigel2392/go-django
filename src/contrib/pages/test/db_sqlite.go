//go:build (!mysql && !postgres && !mariadb && !mysql_local) || (!mysql && !postgres && !mysql_local && !mariadb && !sqlite)

package pages_test

const (
	testPageINSERT       = `INSERT INTO test_pages (description) VALUES (?)`
	testPageUPDATE       = `UPDATE test_pages SET description = ? WHERE id = ?`
	testPageByID         = `SELECT id, description FROM test_pages WHERE id = ?`
	testPageCREATE_TABLE = `CREATE TABLE IF NOT EXISTS test_pages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		description TEXT
	)`
)
