package test

import (
	"encoding/json"
	"fmt"
	srv "myproject/face-recognition"
	"os"
	"path/filepath"
	"testing"
)

func TestConvertStringSliceToImagePathSlice(t *testing.T) {
	path := "/Volumes/Latte/PIC/2026/home/bris/0-face-recognition/convert-folder"
	result, err := srv.GetCollectImage(path)
	if err != nil {
		t.Fatalf("failed to get path collection: %v", err)
	}

	// fmt.Println("result is::: ", result)
	// fmt.Println("total paths are::: ", len(result))

	// convert []string to []ImagePath
	converted := srv.ConvertStringSliceToImagePathSlice(result)

	// fmt.Println("converted result::: ", converted)
	prettyJSON, err := json.MarshalIndent(converted, "", " ")
	if err != nil {
		fmt.Println("Error marshaling to JSON:", err)
		return
	}

	fmt.Println(string(prettyJSON))

}

func TestHandleStoreImageToFolder(t *testing.T) {
	// Create a temporary source directory and file

	srcDir, err := os.MkdirTemp(".", "testsrc")
	if err != nil {
		t.Fatalf("Failed to create temp src dir: %v", err)
	}
	defer os.RemoveAll(srcDir)

	srcFile := filepath.Join(srcDir, "testimage.jpg")
	err = os.WriteFile(srcFile, []byte("fake image data"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a temporary destination directory
	dstDir, err := os.MkdirTemp(".", "testdst")
	if err != nil {
		t.Fatalf("Failed to create temp dst dir: %v", err)
	}
	defer os.RemoveAll(dstDir)

	// Define the person and call the function
	person := srv.Person{
		Name: "TestPerson",
		Image: srv.Image{
			Name: "testimage.jpg",
		},
	}

	err = srv.HandleStoreImageToFolder(person, srcDir, dstDir)
	if err != nil {
		t.Fatalf("HandleStoreImageToFolder failed: %v", err)
	}

	// Check if the file was copied to the destination directory
	dstFile := filepath.Join(dstDir, person.Name, "testimage.jpg")
	if _, err := os.Stat(dstFile); os.IsNotExist(err) {
		t.Errorf("Expected file to exist at %s but it does not", dstFile)
	}
}
