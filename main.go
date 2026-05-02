package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"

	facerecognition "myproject/face-recognition"
	"myproject/face-recognition/repo"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file, reading from environment")
	}

	cfg := &repo.Config{
		Database: repo.DatabaseConfig{
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			Name:     os.Getenv("DB_NAME"),
		},
	}

	db, err := repo.InitDB(cfg)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer db.Close()

	if err := repo.InitSchema(db); err != nil {
		log.Fatalf("schema init: %v", err)
	}

	repo := facerecognition.NewFaceRepo(db)
	log.Println("db connected and schema ready")

	subjects := []string{"phoebe", "vickie"}

	src := "./original-images/all/"

	// Create temp working directory for this test
	workDir, err := os.MkdirTemp(".", "classified")
	if err != nil {
		log.Fatalf("failed to create working temp dir: %v", err)
	}
	// defer os.RemoveAll(workDir)

	dst := filepath.Join(workDir, "dst")

	err = facerecognition.StoreImageToSubjectFolder(context.Background(), repo, subjects, src, dst)

	if err != nil {
		log.Fatalf("failed to copy files: %v", err)
		return
	}

	log.Println("completed")

}
