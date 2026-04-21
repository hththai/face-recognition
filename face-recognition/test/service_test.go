package test

import (
	"encoding/json"
	"fmt"
	srv "myproject/face-recognition"
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
