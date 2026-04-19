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

// Insert into file_subject_image
func (r *faceRepoImpl) InsertFaceAndImage(ctx context.Context, subjectID int64, images []ImageFile) (int64, error) {

	return 0, nil
}
