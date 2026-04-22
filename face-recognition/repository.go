package facerecognition

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
)

type FaceRepository interface {
	InsertFaceSubject(ctx context.Context, subList []string) (int64, error)
	InsertFilePath(ctx context.Context, images []ImageFile) (int64, error)
	InsertFaceAndImage(ctx context.Context, persons []Person) (int64, error)
	GetImagesAndProcess(ctx context.Context, limit int) ([]ImageFile, error)
}

type faceRepoImpl struct {
	db *sql.DB
}

// Make a constructor for the repository
func NewFaceRepo(db *sql.DB) FaceRepository {
	return &faceRepoImpl{db: db}
}

// Insert a list of subjects to database
func (r *faceRepoImpl) InsertFaceSubject(ctx context.Context, subList []string) (int64, error) {
	if len(subList) == 0 {
		return 0, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)

	if err != nil {
		return 0, err
	}

	placeholders := make([]string, len(subList))

	args := make([]interface{}, len(subList))

	for i, name := range subList {
		placeholders[i] = "(?)"
		args[i] = name
	}

	query := fmt.Sprintf("INSERT IGNORE INTO face_subject(name) VALUES %s;", strings.Join(placeholders, ","))

	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return affected, nil
}

// Insert file path into the database - face_image_path (file_path, file_name)
// Can have many folders, sub folders, and files - 2000 files
func (r *faceRepoImpl) InsertFilePath(ctx context.Context, images []ImageFile) (int64, error) {
	if len(images) == 0 {
		return 0, fmt.Errorf("empty images")
	}

	tx, err := r.db.BeginTx(ctx, nil)

	if err != nil {
		return 0, err
	}

	var args []interface{}
	valueStrings := make([]string, 0, len(images))

	for _, image := range images {
		fileName := filepath.Base(image.Path)
		args = append(args, image.Path, fileName)
		valueStrings = append(valueStrings, "(?, ?)")
	}

	query := fmt.Sprintf("INSERT IGNORE INTO face_image_path (file_path, file_name) VALUES %s", strings.Join(valueStrings, ", "))

	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to insert file paths: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	affected, _ := res.RowsAffected()

	return affected, nil
}

func (r *faceRepoImpl) InsertFaceAndImage(ctx context.Context, persons []Person) (int64, error) {

	tx, err := r.db.BeginTx(ctx, nil)

	if err != nil {
		return 0, err
	}

	var args []interface{}
	valueStrings := make([]string, 0, len(persons))

	for _, person := range persons {
		subject := person.Name
		image := person.Image.Name
		args = append(args, subject, image)
		valueStrings = append(valueStrings, "(?, ?)")
	}

	query := fmt.Sprintf("INSERT INTO face_subject_image (subject, file) VALUES %s", strings.Join(valueStrings, ", "))
	// `INSERT INTO face_subject_image (subject, file) VALUES (?,?)`

	res, err := tx.ExecContext(ctx, query, args...)

	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	affected, _ := res.RowsAffected()

	return affected, nil
}

// Test Get dummy first value
// Get and Update status of x image to "processing".
func (r *faceRepoImpl) GetImagesAndProcess(ctx context.Context, limit int) ([]ImageFile, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	var images []ImageFile

	query := `
			SELECT file_path, file_name
			FROM face_image_path 
			WHERE status="pending" 
			LIMIT ?
			FOR UPDATE SKIP LOCKED`

	rows, err := tx.QueryContext(ctx, query, limit)

	if err != nil {
		tx.Rollback()
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var rec ImageFile

		if err := rows.Scan(&rec.Path, &rec.Name); err != nil {
			rows.Close()
			tx.Rollback()
			return nil, err
		}
		images = append(images, rec)
	}

	if err := rows.Err(); err != nil {
		tx.Rollback()
		return nil, err
	}

	// Mark as processing
	updateQuery := `UPDATE face_image_path SET status='processing' WHERE file_name=?`
	for _, image := range images {
		_, err = tx.ExecContext(ctx, updateQuery, image.Name)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return images, nil
}
