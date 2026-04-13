//go:build integration

package test

import (
	"net/http"
	"os"
	"testing"

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
	Image_Path = "/Volumes/Latte/PIC/2026/home/bris/0-face-recognition/convert-folder/DSCF2718.JPG"
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
	assert.Equal(t, "john", person.Name, "Expected name 'John'")
	assert.Equal(t, testImagePath, person.Image.Name, "Expected image path to match")
}
