//go:build integration

package test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"testing"
	"time"

	srv "myproject/face-recognition"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

// MockHandler is a custom http.Handler for testing.
type MockHandler struct {
	StatusCode   int
	ResponseBody []byte
}

func (mh *MockHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rw.WriteHeader(mh.StatusCode)
	rw.Write(mh.ResponseBody)
}

const (
	Image_Path = "/Volumes/Latte/PIC/2026/home/bris/0-face-recognition/convert-folder/DSCF2750.JPG"
)

// TestGetFaceFromImageIntegration tests the GetFaceFromImage function.
func TestGetFaceFromImageIntegration(t *testing.T) {

	// Load .env file
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatalf("Error loading .env file: %v", err)
	}

	// Retrieve environment variables
	os.Getenv("EXADEL_SERVICE_URL")
	os.Getenv("EXADEL_API_KEY")

	// Create a test image file
	testImagePath := Image_Path

	// Call the function under test
	persons := srv.GetFaceFromImage(testImagePath)

	jsonResult, err := json.MarshalIndent(persons, "", "    ")
	if err != nil {
		t.Fatalf("failed to marshal result to JSON: %v", err)
	}
	fmt.Printf("persons:\n%s\n", jsonResult)

	assert.NotEmpty(t, persons, "Expected at least one Person detected")
	for _, person := range persons {
		assert.NotEmpty(t, person.Name, "Expected a non-empty name")
		assert.Equal(t, testImagePath, person.Image.Path, "Expected image path to match")
	}
	assert.Equal(t, "phoebe", persons[0].Name, "Expected first person name 'phoebe'")
}

// Test function
func TestScanFaceFromFolder(t *testing.T) {
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatalf("Error loading .env file: %v", err)
	}

	path := "/Volumes/Latte/PIC/2026/home/bris/0-face-recognition/convert-folder"

	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	startTime := time.Now()

	persons, err := srv.ScanFaceFromFolder(path, "phoebe")

	timeTaken := time.Since(startTime)
	runtime.ReadMemStats(&memAfter)

	assert.NoError(t, err)
	assert.NotNil(t, persons)

	fmt.Printf("ScanFaceFromFolder took %v\n", timeTaken)
	fmt.Printf("Memory used: %.2f MB\n", float64(memAfter.Alloc-memBefore.Alloc)/(1024*1024))
}

// Test GetFaceSubjects save to database
func TestGetFaceSubjects(t *testing.T) {
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatalf("Error loading .env file: %v", err)
	}

	result, err := srv.GetFaceSubjects()

	if err != nil {
		t.Fatalf("failed to get subjects: %v", err)
	}

	fmt.Printf("result is::: %s\n", result)

	// Convert result from &[]string to []string
	inputSubList := result.PersonName

	fmt.Printf("converted to list string: %v\n", inputSubList)

	// DB connection established
	db := setupTestDB(t)
	defer db.Close()

	repoSvc := srv.NewFaceRepo(db)

	affected, err := repoSvc.InsertFaceSubject(context.Background(), inputSubList)

	if err != nil {
		t.Fatalf("failed to insert subject name: %v", err)
	}

	fmt.Printf("Inserted %d rows\n", affected)

}

func TestInsertFilePath(t *testing.T) {
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatalf("Error loading .env file: %v", err)
	}

	// DB connection established
	db := setupTestDB(t)
	defer db.Close()

	// Define example []ImageFile
	imageFiles := []srv.ImageFile{
		{
			Path: "/Volumes/Latte/PIC/2026/home/bris/0-face-recognition/convert-folder/DSCF2998.JPG",
		},
		{
			Path: "/Volumes/Latte/PIC/2026/home/bris/0-face-recognition/convert-folder/DSCF2713.JPG",
		},
		{
			Path: "/Volumes/Latte/PIC/2026/home/bris/0-face-recognition/convert-folder/DSCF2714.JPG",
		},
	}

	repo := srv.NewFaceRepo(db)
	affected, err := repo.InsertFilePath(context.Background(), imageFiles)

	if err != nil {
		t.Fatalf("failed to insert File Path: %v", err)
	}

	fmt.Printf("insert success %d", affected)
}

// Test function StoreFilePaths(ctx context.Context, filePath []ImageFile, repo FaceRepository) error
func TestInsertFilePathByService(t *testing.T) {
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatalf("Error loading .env file: %v", err)
	}

	// DB connection established
	db := setupTestDB(t)
	defer db.Close()

	// Call direct service function
	path := "/Volumes/Latte/PIC/2026/home/bris/0-face-recognition/convert-folder/"
	imagePaths, err := srv.GetCollectImage(path)
	if err != nil {
		t.Fatalf("failed to get path collection: %v", err)
	}

	// Convert to ImagePath
	imageFiles := srv.ConvertStringSliceToImagePathSlice(imagePaths)

	repo := srv.NewFaceRepo(db)

	affected, err := srv.StoreFilePaths(context.Background(), imageFiles, repo)

	if err != nil {
		t.Fatalf("failed to insert File Path: %v", err)
	}

	fmt.Printf("insert success %d\n", affected)
}

// Test insert face and image
func TestInsertFaceAndImage(t *testing.T) {
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatalf("Error loading .env file: %v", err)
	}

	// DB connection established
	db := setupTestDB(t)
	defer db.Close()

	subject := srv.Subject{Subject: "phoebe"}
	img := "DSCF2778.JPG"

	repo := srv.NewFaceRepo(db)

	affected, err := repo.InsertFaceAndImage(context.Background(), subject, img)

	if err != nil {
		t.Fatalf("failed to insert %v", err)
	}

	fmt.Printf("insert success %d\n", affected)

}

// Test get dummy 1 path
func TestGetImagesAndProcess(t *testing.T) {
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatalf("Error loading .env file: %v", err)
	}

	// DB connection established
	db := setupTestDB(t)
	defer db.Close()

	repo := srv.NewFaceRepo(db)

	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	startTime := time.Now()

	filePath, err := repo.GetImagesAndProcess(context.Background(), 50)

	timeTaken := time.Since(startTime)
	runtime.ReadMemStats(&memAfter)

	if err != nil {
		t.Fatalf("failed to get first path: %v", err)
	}

	fmt.Printf("GetFirstImagePath took %v\n", timeTaken)
	fmt.Printf("Memory used: %.2f MB\n", float64(memAfter.Alloc-memBefore.Alloc)/(1024*1024))
	fmt.Println("Value path is >>> ", filePath)
}

// Test Get Faces from a list of []ImagePath repo
func TestGetFacesService(t *testing.T) {

	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatalf("Error loading .env file: %v", err)
	}

	// DB connection established
	db := setupTestDB(t)
	defer db.Close()
	repo := srv.NewFaceRepo(db)

	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	startTime := time.Now()

	persons, err := srv.GetFaces(context.Background(), repo, 50)
	if err != nil {
		t.Fatalf("failed to get faces: %v", err)
	}

	jsonResult, err := json.MarshalIndent(persons, "", "    ")
	if err != nil {
		t.Fatalf("failed to marshal result to JSON: %v", err)
	}
	fmt.Printf("GetFacesService result:\n%s\n", jsonResult)

	timeTaken := time.Since(startTime)
	runtime.ReadMemStats(&memAfter)

	fmt.Printf("\u001B[32mGetPathService took %v\n\u001B[0m", timeTaken)
	fmt.Printf("\u001B[32mMemory used: %.2f MB\n\u001B[0m", float64(memAfter.Alloc-memBefore.Alloc)/(1024*1024))

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
