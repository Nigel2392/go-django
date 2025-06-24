//go:build !mysql && !mysql_local && !mariadb && !sqlite && postgres

package pages_test

const (
	testPageINSERT       = `INSERT INTO test_pages (description) VALUES ($1)`
	testPageUPDATE       = `UPDATE test_pages SET description = $1 WHERE id = $2`
	testPageByID         = `SELECT id, description FROM test_pages WHERE id = $1`
	testPageCREATE_TABLE = `CREATE TABLE IF NOT EXISTS test_pages (
	 	id  SERIAL PRIMARY KEY,
	 	description  VARCHAR(255) NOT NULL
	)`
)
