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
