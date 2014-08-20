package util

import (
	"database/sql"
	"fmt"

	"github.com/opentarock/service-user-management/util/logutil"
)

func Prepare(db *sql.DB, statements map[string]*sql.Stmt, name, query string) error {
	if _, ok := statements[name]; !ok {
		stmt, err := db.Prepare(query)
		if err != nil {
			logutil.ErrorFatal(fmt.Sprintf("Error preparing statement %s", name), err)
		}
		statements[name] = stmt
		return nil
	}
	panic(fmt.Sprintf("Statement %s already exists", name))
}

func QueryRow(statements map[string]*sql.Stmt, name string, args ...interface{}) *sql.Row {
	if stmt, ok := statements[name]; ok {
		return stmt.QueryRow(args...)
	}
	panic(fmt.Sprintf("QueryRow statement not found: %s", name))
}

func Exec(statements map[string]*sql.Stmt, name string, args ...interface{}) (sql.Result, error) {
	if stmt, ok := statements[name]; ok {
		return stmt.Exec(args...)
	}
	panic(fmt.Sprintf("Exec statement not found: %s", name))
}
