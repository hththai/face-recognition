package facerecognition

type Person struct {
	Name  string `json:"name"`
	Image Image  `json:"image"`
}

type Image struct {
	Path string
	Name string
}

// Define the minimal structs needed to extract Subject information
type Response struct {
	StatusCode int
	Result     []Result `json:"result"`
}

type Result struct {
	Subjects []Subject `json:"subjects"`
}

type Subject struct {
	Subject    string  `json:"subject"`
	Similarity float64 `json:"similarity"`
}

type scanResult struct {
	person   *Person
	filePath string
}

type SubjectResponse struct {
	PersonName []string `json:"subjects"`
}

type ImageFile struct {
	Name string `json:"filename"`
	Path string `json:"filepath"`
}
