package facerecognition

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type FaceRepository interface {
	InsertFaceSubject(ctx context.Context, subList []string) (int64, error)
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

// Insert file path into the database.
func InsertFilePath(ctx context.Context, db *sql.DB, filePaths []string) error {
	return nil
}
