package repo

import "database/sql"

func InitSchema(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS face_subject (
			id   INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(128) NOT NULL UNIQUE
		)`,
		`CREATE TABLE IF NOT EXISTS face_image_path (
			id         INT PRIMARY KEY AUTO_INCREMENT NOT NULL,
			file_path  VARCHAR(255) NOT NULL,
			file_name  VARCHAR(128) NOT NULL UNIQUE,
			status     ENUM('pending','processing','error','completed') DEFAULT 'pending',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS face_subject_image (
			subject VARCHAR(128) NOT NULL,
			file    VARCHAR(128) NOT NULL,
			PRIMARY KEY (subject, file),
			FOREIGN KEY (subject) REFERENCES face_subject(name)          ON DELETE CASCADE,
			FOREIGN KEY (file)    REFERENCES face_image_path(file_name)  ON DELETE CASCADE
		)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
