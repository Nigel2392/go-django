//go:build !sqlite && !postgres && (mysql_local || mysql || mariadb)

package pages_test

const (
	testPageINSERT       = `INSERT INTO test_pages (description) VALUES (?)`
	testPageUPDATE       = `UPDATE test_pages SET description = ? WHERE id = ?`
	testPageByID         = `SELECT id, description FROM test_pages WHERE id = ?`
	testPageCREATE_TABLE = `CREATE TABLE IF NOT EXISTS test_pages (
		id INT AUTO_INCREMENT PRIMARY KEY,
		description VARCHAR(255) NOT NULL
	)`
)
