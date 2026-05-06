package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	facerecognition "myproject/face-recognition"
	grpcserver "myproject/face-recognition/grpc/server"
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

	faceRepo := facerecognition.NewFaceRepo(db)
	log.Println("db connected and schema ready")

	// --- batch job (commented out while gRPC server is active) ---
	// subjects := []string{"phoebe", "vickie"}
	// src := "./original-images/all/"
	// workDir, err := os.MkdirTemp(".", "classified")
	// if err != nil {
	// 	log.Fatalf("failed to create working temp dir: %v", err)
	// }
	// dst := filepath.Join(workDir, "dst")
	// if err := facerecognition.StoreImageToSubjectFolder(context.Background(), faceRepo, subjects, src, dst); err != nil {
	// 	log.Fatalf("failed to copy files: %v", err)
	// }
	// log.Println("completed")

	addr := os.Getenv("GRPC_ADDR")
	if addr == "" {
		addr = ":50051"
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		errCh <- grpcserver.Run(addr, faceRepo)
	}()

	select {
	case err := <-errCh:
		log.Fatalf("gRPC server error: %v", err)
	case sig := <-stop:
		log.Printf("received %s, shutting down", sig)
	}
}
