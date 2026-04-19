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

	dropTableQuery := `DROP TABLE IF EXISTS face_subject_image, face_subject, face_image_path; `

	_, err := db.ExecContext(ctx, dropTableQuery)

	if err != nil {
		t.Fatalf("failed to drop table: %v", err)
	}

	// Create table subject person
	tableCreationQuery := "CREATE TABLE IF NOT EXISTS `face_subject` (`id` INT AUTO_INCREMENT PRIMARY KEY, `name` VARCHAR(128) NOT NULL UNIQUE)"

	_, err = db.ExecContext(ctx, tableCreationQuery)

	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Create table file image path
	tblFileQuery := "CREATE TABLE IF NOT EXISTS `face_image_path` (`id` INT PRIMARY KEY AUTO_INCREMENT NOT NULL, `file_path` VARCHAR(255) NOT NULL, `file_name` VARCHAR(128) NOT NULL UNIQUE)"

	_, err = db.ExecContext(ctx, tblFileQuery)

	if err != nil {
		t.Fatalf("faied to create table: %v", err)
	}

	// Create table many to many subjects and faces
	tblFileToImageQuery := `
		CREATE TABLE face_subject_image (
				subject VARCHAR(128) NOT NULL,
				file VARCHAR(128) NOT NULL,
				primary key (subject, file),
				foreign key (subject) references face_subject(name) on delete cascade,
				foreign key (file) references face_image_path(file_name) on delete cascade
  			)`

	_, err = db.ExecContext(ctx, tblFileToImageQuery)

	if err != nil {
		t.Fatalf("faied to create table: %v", err)
	}

	//
	// Insert Subject data
	// insertSubjectQuery := "INSERT INTO `face_subject` (name) VALUES ('phoebe'),('john'),('vickie')"

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
