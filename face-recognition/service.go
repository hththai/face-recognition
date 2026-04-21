package facerecognition

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Send Request to the exadel service.
// If return Ok, store metadata to database as Person
// schema of return is Subject.
// Map Subject to Person
func GetFaceFromImage(path string) *Person {
	if path == "" {
		return nil
	}

	// Open the image file
	file, err := os.Open(path)
	if err != nil {
		log.Printf("Error opening file: %v", err)
		return nil
	}
	defer file.Close()

	// Create a new multipart writer to encode form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create a form file part and copy the image content into it
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		log.Printf("Error creating form file part: %v", err)
		return nil
	}
	_, err = io.Copy(part, file)
	if err != nil {
		log.Printf("Error copying file content: %v", err)
		return nil
	}

	// Close the multipart writer
	err = writer.Close()
	if err != nil {
		log.Printf("Error closing multipart writer: %v", err)
		return nil
	}

	// Create a new HTTP request to the Exadel service
	endpoint := os.Getenv("EXADEL_SERVICE_URL")
	if endpoint == "" {
		log.Printf("EXADEL_SERVICE_URL is not set")
		return nil
	}
	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return nil
	}

	// Add necessary headers
	req.Header.Add("x-api-key", os.Getenv("EXADEL_API_KEY"))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send the request and handle the response
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request: %v", err)
		return nil
	}
	defer resp.Body.Close()

	var response Response
	// Read the response body into a string for logging
	responseBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response: %v", err)
		return nil
	}
	defer resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewBuffer(responseBodyBytes))

	log.Printf("Response Body: %s", responseBodyBytes)

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Printf("Error decoding response: %v", err)
		return nil
	}

	person := &Person{
		Image: Image{
			Name: path,
		},
	}

	// Print the extracted Subject data
	for _, result := range response.Result {
		for _, subject := range result.Subjects {
			log.Println("Subject:", subject.Subject, "Similarity:", subject.Similarity)

			person.Name = subject.Subject
		}
	}

	return person
}

func collectImagePaths(path string) ([]string, error) {
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	entries, err := dir.ReadDir(-1)
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		ext := strings.ToUpper(filepath.Ext(entry.Name()))
		if ext != ".JPG" && ext != ".JPEG" && ext != ".PNG" {
			continue
		}
		paths = append(paths, filepath.Join(path, entry.Name()))
	}
	return paths, nil
}

func scanWorkers(imagePaths []string, expectPerson string) <-chan scanResult {
	const workers = 2
	jobs := make(chan string, len(imagePaths))
	results := make(chan scanResult, len(imagePaths))

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fp := range jobs {
				p := GetFaceFromImage(fp)
				if p != nil && p.Name == expectPerson {
					results <- scanResult{p, fp}
				}
			}
		}()
	}

	for _, fp := range imagePaths {
		jobs <- fp
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

// List subjects and store persons as list to run compare.
func GetFaceSubjects() (*SubjectResponse, error) {
	// Create a new HTTP request to the Exadel service
	endpoint := os.Getenv("EXADEL_ENDPOINT")
	if endpoint == "" {
		log.Printf("EXADEL_ENDPOINT is not set")
		return nil, nil
	}

	req, err := http.NewRequest("GET", endpoint+"/recognition/subjects", nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return nil, err
	}

	// Add necessary headers
	req.Header.Add("x-api-key", os.Getenv("EXADEL_API_KEY"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var response SubjectResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Printf("Error decoding response: %v", err)
		return nil, err
	}

	return &response, nil
}

// Scan provided folder and record the image person related to the subject
// From provided folder path, the function will run GetFaceFromImage.
// If it matches expected person, it will store the file name of image into a file.txt as output.
func ScanFaceFromFolder(path string, expectPerson string) ([]*Person, error) {
	imagePaths, err := collectImagePaths(path)
	if err != nil {
		return nil, err
	}

	// Temporary write to file. It will write to database
	f, err := os.OpenFile("file.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var persons []*Person
	for r := range scanWorkers(imagePaths, expectPerson) {
		persons = append(persons, r.person)
		_, err = f.WriteString(r.filePath + "\n")
		if err != nil {
			return nil, err
		}
	}

	return persons, nil
}

// ... rest of code ...

// Dummy for test case
func GetCollectImage(path string) ([]string, error) {
	return collectImagePaths(path)
}

func ConvertStringSliceToImagePathSlice(strSlice []string) []ImageFile {
	var imagePathSlice []ImageFile
	for _, str := range strSlice {

		// get name of file
		name := filepath.Base(str)

		imagePathSlice = append(imagePathSlice, ImageFile{name, str})
	}
	return imagePathSlice
}

// Import files location to database.
// using collect image paths function.
func StoreFilePaths(ctx context.Context, filePath []ImageFile, repo FaceRepository) (int64, error) {
	if len(filePath) == 0 {
		return 0, fmt.Errorf("invalid path input")
	}

	affected, err := repo.InsertFilePath(ctx, filePath)

	if err != nil {
		return 0, fmt.Errorf("failed to store file path %v", err)
	}

	return affected, nil
}

// Call API and get Face from image
func GetFaces(ctx context.Context, repo FaceRepository) string {

	imageRecords, err := repo.GetFirstImage(ctx)

	if err != nil {
		return ""
	}

	// Call API
	person := GetFaceFromImage(imageRecords[0].Path)

	fmt.Println("person is::: ", person)

	return imageRecords[0].Path
	// return ""
}
