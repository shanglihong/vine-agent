package sqlite

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"vine-agent/utils"

	_ "modernc.org/sqlite"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	var err error
	testDB, err = sql.Open("sqlite", ":memory:")
	if err != nil {
		panic("open in-memory sqlite: " + err.Error())
	}

	root := utils.FindProjectRoot()
	var sqlPath string
	if root != "" {
		sqlPath = filepath.Join(root, "scripts", "sqlite_memory.sql")
	} else {
		sqlPath = "../../../scripts/sqlite_memory.sql"
	}

	content, err := os.ReadFile(sqlPath)
	if err != nil {
		testDB.Close()
		panic("read schema file: " + err.Error())
	}

	if _, err := testDB.Exec(string(content)); err != nil {
		testDB.Close()
		panic("apply schema: " + err.Error())
	}

	code := m.Run()
	testDB.Close()
	os.Exit(code)
}
