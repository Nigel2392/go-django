package django

import (
	"database/sql"
	"flag"
	"os"
	"slices"
	"strings"

	"github.com/Nigel2392/go-django/src/core/command"
)

func makeQuery(query string, m command.Manager, db *sql.DB) error {
	rows, err := db.Query(query)
	if err != nil {
		m.Logf("Error executing query: %s\n", err.Error())
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		m.Logf("Error retrieving columns: %s\n", err.Error())
		return err
	}

	var values = make([]interface{}, len(columns))
	var valuePtrs = make([]interface{}, len(columns))
	for i := range columns {
		valuePtrs[i] = &values[i]
	}

	var hadRows bool
	for rows.Next() {

		if hadRows {
			m.Log("---")
		}

		err = rows.Scan(valuePtrs...)
		if err != nil {
			m.Logf("Error scanning row: %s\n", err.Error())
			continue
		}

		for i, col := range columns {
			m.Logf("%s: %v\n", col, values[i])
		}

		hadRows = true
	}

	return nil
}

var sqlShellCommand = &command.Cmd[string]{
	ID:   "sql",
	Desc: "Run a SQL shell",
	FlagFunc: func(m command.Manager, stored *string, f *flag.FlagSet) error {
		f.StringVar(stored, "q", "", "Query to run without entering the shell")
		return nil
	},
	Execute: func(m command.Manager, stored string, args []string) error {

		var db, ok = ConfigGetOK[*sql.DB](Global.Settings, APPVAR_DATABASE)
		if !ok {
			m.Log("No database connection found")
			os.Exit(1)
			return nil
		}

		if stored != "" {
			if err := makeQuery(stored, m, db); err != nil {
				os.Exit(1)
				return nil
			}
			os.Exit(0)
			return nil
		}

		m.Log("Enter 'exit' or 'quit' to exit the shell")
		for {
			var query, err = m.Input("sql> ")
			if err != nil {
				m.Logf("Error reading input: %s\n", err.Error())
				os.Exit(1)
				return nil
			}

			if slices.Contains([]string{"exit", "quit"}, strings.ToLower(query)) {
				break
			}

			makeQuery(query, m, db)
		}

		os.Exit(0)
		return nil
	},
}
