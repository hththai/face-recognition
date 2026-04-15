//go:build integration

package test

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"testing"
	"time"

	srv "myproject/face-recognition"

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
	Image_Path = "/Volumes/Latte/PIC/2026/home/bris/0-face-recognition/convert-folder/DSCF2998.JPG"
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
	person := srv.GetFaceFromImage(testImagePath)

	// Assert the results
	assert.NotNil(t, person, "Expected a non-nil Person object")
	assert.Equal(t, "phoebe", person.Name, "Expected name 'Phoebe'")
	assert.Equal(t, testImagePath, person.Image.Name, "Expected image path to match")
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
	fmt.Printf("Memory used: %d bytes\n", memAfter.Alloc-memBefore.Alloc)
}
