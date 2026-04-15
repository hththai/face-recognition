package facerecognition

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Define the minimal structs needed to extract Subject information
type Response struct {
	Result []Result `json:"result"`
}

type Result struct {
	Subjects []Subject `json:"subjects"`
}

type Subject struct {
	Subject    string  `json:"subject"`
	Similarity float64 `json:"similarity"`
}

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

// Scan provided folder and record the image person related to the subject
// From provided folder path, the function will run GetFaceFromImage.
// If it matches expected person, it will store the file name of image into a file.txt as output.
func ScanFaceFromFolder(path string, expectPerson string) ([]*Person, error) {
	var persons []*Person

	// Open the directory.
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	// Read all files in the directory.
	entries, err := dir.ReadDir(-1)
	if err != nil {
		return nil, err
	}

	// Open output file once, outside the loop.
	f, err := os.OpenFile("file.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		ext := strings.ToUpper(filepath.Ext(entry.Name()))
		if ext != ".JPG" && ext != ".JPEG" && ext != ".PNG" {
			continue
		}

		// Get the full file path.
		filePath := filepath.Join(path, entry.Name())

		// Get faces from the image.
		person := GetFaceFromImage(filePath)

		if person != nil {

			// if result person.Name matched the input parameter expectPerson
			if person.Name != expectPerson {
				continue
			}

			persons = append(persons, person)

			_, err = f.WriteString(filePath + "\n")
			if err != nil {
				return nil, err
			}
		}
	}

	return persons, nil
}

// ... rest of code ...
