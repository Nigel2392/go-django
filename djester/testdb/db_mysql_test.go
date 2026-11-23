//go:build !sqlite && !postgres && !mariadb && !mysql_local && mysql

package testdb

import "testing"

func TestMySQLFixtureIsolation(t *testing.T) {
	_, first := open()
	defer first.Close()

	_, second := open()
	defer second.Close()

	ctx := t.Context()
	var firstName, secondName string
	if err := first.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&firstName); err != nil {
		t.Fatal(err)
	}

	if err := second.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&secondName); err != nil {
		t.Fatal(err)
	}

	if firstName == secondName || firstName == "django-test" || secondName == "django-test" {
		t.Fatalf("fixtures share a database: %q, %q", firstName, secondName)
	}

	if _, err := first.ExecContext(ctx, "CREATE TABLE fixture (id INT PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}

	if _, err := second.ExecContext(ctx, "CREATE TABLE fixture (code VARCHAR(20) PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}

	if _, err := first.ExecContext(ctx, "INSERT INTO fixture (id) VALUES (1)"); err != nil {
		t.Fatal(err)
	}

	if _, err := second.ExecContext(ctx, "INSERT INTO fixture (code) VALUES ('second')"); err != nil {
		t.Fatal(err)
	}

	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	if err := first.Close(); err != nil {
		t.Fatalf("repeated close failed: %v", err)
	}

	var remaining int
	if err := second.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name=?", firstName).Scan(&remaining); err != nil {
		t.Fatal(err)
	}

	if remaining != 0 {
		t.Fatal("closed fixture database was not removed")
	}

	var code string
	if err := second.QueryRowContext(ctx, "SELECT code FROM fixture").Scan(&code); err != nil || code != "second" {
		t.Fatalf("closing one fixture changed the other: %q, %v", code, err)
	}
}
