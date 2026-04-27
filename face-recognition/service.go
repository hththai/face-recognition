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
func GetFaceFromImage(path string) []*Person {
	if path == "" {
		return nil
	}

	response, err := recognizeFaces(path)
	if err != nil {
		log.Printf("Error recognizing faces: %v", err)
		return nil
	}

	image := Image{Name: filepath.Base(path), Path: path}
	return distinctPersons(response, image)
}

func recognizeFaces(path string) (*Response, error) {
	body, contentType, err := buildMultipartBody(path)
	if err != nil {
		return nil, err
	}

	endpoint := os.Getenv("EXADEL_SERVICE_URL")
	if endpoint == "" {
		return nil, fmt.Errorf("EXADEL_SERVICE_URL is not set")
	}

	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Add("x-api-key", os.Getenv("EXADEL_API_KEY"))
	req.Header.Set("Content-Type", contentType)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	// log.Printf("Response Body: %s", data)

	var response Response
	if err = json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	response.StatusCode = resp.StatusCode
	return &response, nil
}

func buildMultipartBody(path string) (*bytes.Buffer, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return nil, "", fmt.Errorf("creating form file part: %w", err)
	}
	if _, err = io.Copy(part, file); err != nil {
		return nil, "", fmt.Errorf("copying file content: %w", err)
	}
	if err = writer.Close(); err != nil {
		return nil, "", fmt.Errorf("closing multipart writer: %w", err)
	}
	return body, writer.FormDataContentType(), nil
}

func distinctPersons(response *Response, image Image) []*Person {
	if response.StatusCode == http.StatusBadRequest {
		return []*Person{{
			Name:  "nobody",
			Image: image,
		}}
	}

	seen := make(map[string]float64)
	for _, result := range response.Result {
		best := bestSubject(result.Subjects)
		if best == nil {
			continue
		}
		if sim, exists := seen[best.Subject]; !exists || best.Similarity > sim {
			seen[best.Subject] = best.Similarity
		}
	}

	persons := make([]*Person, 0, len(seen))
	for name := range seen {
		// log.Printf("Subject: %s  Similarity: %.4f", name, sim)
		persons = append(persons, &Person{Name: name, Image: image})
	}
	return persons
}

func bestSubject(subjects []Subject) *Subject {
	var best *Subject
	for i := range subjects {
		if best == nil || subjects[i].Similarity > best.Similarity {
			best = &subjects[i]
		}
	}
	return best
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
		wg.Go(func() {
			for fp := range jobs {
				for _, p := range GetFaceFromImage(fp) {
					if p.Name == expectPerson {
						results <- scanResult{p, fp}
					}
				}
			}
		})
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
// Repo function name GetImagesAndProcess return []ImageFile{Name, Path}
// This service do is from each Path, it will send request by GetFaceFromImage(path string) and return *Person.
// Each ImageFile in []ImageFile will send request and return the result.
// Return all result as json response, or object response.
func GetFaces(ctx context.Context, repo FaceRepository, limit int) ([]*Person, []string, error) {

	const workers = 5

	imageRecords, err := repo.GetImagesAndProcess(ctx, limit)
	if err != nil {
		return nil, nil, err
	}

	fileNames := make([]string, len(imageRecords))
	for i, img := range imageRecords {
		fileNames[i] = img.Name
	}

	// *** Add work group ***
	jobs := make(chan ImageFile, len(imageRecords))
	results := make(chan []*Person, len(imageRecords))

	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done() // Reduce worker
			for img := range jobs {
				if ctx.Err() != nil {
					return
				}
				results <- GetFaceFromImage(img.Path)
			}
		}()
	}

	for _, img := range imageRecords {
		jobs <- img
	}
	close(jobs)

	// Close wg after all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()
	// -------------

	var people []*Person
	// for _, imageRecord := range imageRecords {
	// 	persons := GetFaceFromImage(imageRecord.Path)
	// 	// log.Printf("image %s: %d face(s) detected", imageRecord.Name, len(persons))
	// 	people = append(people, persons...)
	// }

	for persons := range results {
		people = append(people, persons...)
	}

	return people, fileNames, nil
}

// from name files, subject, and path file source.
// Copy the image to folder which is the name of subject if not exists.
// source file is /.../home/ and file name id DSCF2999.JPG,
// for each file name the funciton will copy it to destination path and folder on the subject name
// destination path as parameter /.../home/subjectname/DSCF2999.JPG
func HandleStoreImageToFolder(person Person, src, dst string) error {
	srcImage := filepath.Join(src, person.Image.Name)

	dstFolder := filepath.Join(dst, person.Name)

	if err := os.MkdirAll(dstFolder, 0755); err != nil {
		return fmt.Errorf("failed to create destination folder: %w", err)
	}
	dstImage := filepath.Join(dstFolder, person.Image.Name)

	if err := copyFile(srcImage, dstImage); err != nil {
		return fmt.Errorf("failed to copy image %s: %w", person.Image.Name, err)
	}
	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create the destination directory if it doesn't exist
	err = os.MkdirAll(filepath.Dir(dst), 0755)
	if err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	err = destFile.Sync()
	if err != nil {
		return err
	}

	return nil
}
