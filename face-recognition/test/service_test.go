//go:build integration

package test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	srv "myproject/face-recognition"

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
	Image_Path = "/Volumes/Latte/PIC/2026/home/bris/0-face-recognition/convert-folder/DSCF2712.JPG"
)

// TestGetFaceFromImageIntegration tests the GetFaceFromImage function.
func TestGetFaceFromImageIntegration(t *testing.T) {
	// Create a mock HTTP server
	mockServer := httptest.NewServer(&MockHandler{
		StatusCode:   http.StatusOK,
		ResponseBody: []byte(`{"subject": "John"}`),
	})
	defer mockServer.Close()

	// Set the mock server URL for testing
	os.Setenv("EXADEL_SERVICE_URL", mockServer.URL)
	defer os.Unsetenv("EXADEL_SERVICE_URL")

	// Create a test image file
	testImagePath := Image_Path
	testImageFile, err := os.Create(testImagePath)
	if err != nil {
		t.Fatalf("Failed to create test image file: %v", err)
	}
	defer os.Remove(testImagePath)
	defer testImageFile.Close()

	// Write some dummy image data
	testImageData := []byte("dummy image data")
	_, err = testImageFile.Write(testImageData)
	if err != nil {
		t.Fatalf("Failed to write test image data: %v", err)
	}

	// Call the function under test
	person := srv.GetFaceFromImage(testImagePath)

	// Assert the results
	assert.NotNil(t, person, "Expected a non-nil Person object")
	assert.Equal(t, "John", person.Name, "Expected name 'John'")
	assert.Equal(t, testImagePath, person.Image.Name, "Expected image path to match")
}
