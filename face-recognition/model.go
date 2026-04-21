package facerecognition

type Person struct {
	Name  string
	Image Image
}

type Image struct {
	Name string
}

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
