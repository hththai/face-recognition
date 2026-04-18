//go:build integration

package repo

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func TestConnection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create table
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	// Drop table
	_, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS `face_subject`")
	if err != nil {
		t.Fatalf("Failed to drop table: %v", err)
	}

	// Create table
	tableCreationQuery := "CREATE TABLE IF NOT EXISTS `face_subject` (`id` INT AUTO_INCREMENT PRIMARY KEY, `name` VARCHAR(128) NOT NULL UNIQUE)"

	_, err = db.ExecContext(ctx, tableCreationQuery)

	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Insert Subject data
	//insertSubjectQuery := "INSERT INTO `face_subject` (name) VALUES ('phoebe'),('john'),('vickie')"

	// _, err = db.ExecContext(ctx, insertSubjectQuery)
	// if err != nil {
	// 	t.Fatalf("failed to insert items: %v", err)
	// }

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
