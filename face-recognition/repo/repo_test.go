//go:build integration

package repo

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

func TestConnection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	connStr := os.Getenv("TEST_DB_DSN")
	if connStr == "" {
		connStr = "testuser:password@tcp(127.0.0.1:3306)/testdb?parseTime=true"
	}

	db, err := sql.Open("mysql", connStr)
	if err != nil {
		t.Fatalf("failed to connect to DB: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping DB: %v", err)
	}

	return db
}
