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
	InsertFilePath(ctx context.Context, images []ImageFile) error
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
// TODO: handle the same file name
func (r *faceRepoImpl) InsertFilePath(ctx context.Context, images []ImageFile) error {
	if len(images) == 0 {
		return fmt.Errorf("no input file")
	}

	tx, err := r.db.BeginTx(ctx, nil)

	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	for _, image := range images {
		fileName := filepath.Base(image.Path)
		_, err := tx.ExecContext(ctx, "INSERT INTO face_image_path (file_path, file_name) VALUES (?, ?)", image.Path, fileName)

		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert file path: %w", err)
		}
	}

	err = tx.Commit()

	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
