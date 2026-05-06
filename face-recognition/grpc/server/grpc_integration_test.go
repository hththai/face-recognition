//go:build integration

package server

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"testing"
	"time"

	facerecognition "myproject/face-recognition"
	pb "myproject/face-recognition/grpc"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const (
	integrationImagePath = "/Volumes/Latte/PIC/2026/home/bris/0-face-recognition/convert-folder/DSCF2733.JPG"
)

// setupIntegrationClient loads .env, connects to real DB, and boots an in-process gRPC server.
func setupIntegrationClient(t *testing.T) (pb.FaceRecognitionServiceClient, func()) {
	t.Helper()

	if err := godotenv.Load("../../../.env"); err != nil {
		t.Fatalf("load .env: %v", err)
	}

	db := setupIntegrationDB(t)
	repo := facerecognition.NewFaceRepo(db)

	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	pb.RegisterFaceRecognitionServiceServer(s, newFaceServer(repo))
	go s.Serve(lis) //nolint:errcheck

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufnet: %v", err)
	}

	return pb.NewFaceRecognitionServiceClient(conn), func() {
		conn.Close()
		s.Stop()
		db.Close()
	}
}

func setupIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = "testuser:password@tcp(127.0.0.1:3306)/testdb?parseTime=true"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping db: %v", err)
	}
	return db
}

// TestGRPCRecognizeImage sends a real image to CompreFace via gRPC and verifies persons are returned.
func TestGRPCRecognizeImage(t *testing.T) {
	client, cleanup := setupIntegrationClient(t)
	defer cleanup()

	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	start := time.Now()

	resp, err := client.RecognizeImage(context.Background(), &pb.RecognizeRequest{
		FilePath: integrationImagePath,
	})

	elapsed := time.Since(start)
	runtime.ReadMemStats(&memAfter)

	if err != nil {
		t.Fatalf("RecognizeImage: %v", err)
	}

	fmt.Printf("RecognizeImage took %v\n", elapsed)
	fmt.Printf("Memory used: %.2f MB\n", float64(memAfter.Alloc-memBefore.Alloc)/(1024*1024))
	fmt.Printf("Persons detected: %d\n", len(resp.Persons))
	for _, p := range resp.Persons {
		fmt.Printf("  subject=%s  file=%s\n", p.Subject, p.FileName)
	}

	if len(resp.Persons) == 0 {
		t.Error("expected at least one person detected")
	}
}

// TestGRPCGetSubjectFiles queries a known subject from the real DB via gRPC.
func TestGRPCGetSubjectFiles(t *testing.T) {
	client, cleanup := setupIntegrationClient(t)
	defer cleanup()

	start := time.Now()

	resp, err := client.GetSubjectFiles(context.Background(), &pb.SubjectRequest{
		Subject: "phoebe",
	})

	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("GetSubjectFiles: %v", err)
	}

	fmt.Printf("GetSubjectFiles took %v\n", elapsed)
	fmt.Printf("Subject: %s — %d file(s)\n", resp.Subject, len(resp.Filenames))
	for _, f := range resp.Filenames {
		fmt.Printf("  %s\n", f)
	}

	if resp.Subject != "phoebe" {
		t.Errorf("expected subject phoebe, got %s", resp.Subject)
	}
}

// TestGRPCRecognizeBatch streams multiple image paths and measures throughput vs unary.
func TestGRPCRecognizeBatch(t *testing.T) {
	client, cleanup := setupIntegrationClient(t)
	defer cleanup()

	imagePaths := []string{
		"/Volumes/Latte/PIC/2026/home/bris/0-face-recognition/convert-folder/DSCF2733.JPG",
		"/Volumes/Latte/PIC/2026/home/bris/0-face-recognition/convert-folder/DSCF2778.JPG",
		"/Volumes/Latte/PIC/2026/home/bris/0-face-recognition/convert-folder/DSCF2750.JPG",
	}

	stream, err := client.RecognizeBatch(context.Background())
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	start := time.Now()

	for _, p := range imagePaths {
		if err := stream.Send(&pb.RecognizeRequest{FilePath: p}); err != nil {
			t.Fatalf("send %s: %v", p, err)
		}
	}
	stream.CloseSend()

	var totalPersons int
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("recv: %v", err)
		}
		totalPersons += len(resp.Persons)
		for _, p := range resp.Persons {
			fmt.Printf("  subject=%s  file=%s\n", p.Subject, p.FileName)
		}
	}

	elapsed := time.Since(start)
	runtime.ReadMemStats(&memAfter)

	fmt.Printf("RecognizeBatch: %d image(s), %d person(s) total\n", len(imagePaths), totalPersons)
	fmt.Printf("\033[32mTotal time: %v\033[0m\n", elapsed)
	fmt.Printf("\033[32mMemory used: %.2f MB\033[0m\n", float64(memAfter.Alloc-memBefore.Alloc)/(1024*1024))
}
