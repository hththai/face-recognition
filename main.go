package main

import (
	"fmt"
	convertservice "myproject/convert"
	"os"
)

func main() {
	if err := convertservice.ConvertFolder("original-images", "convert-folder"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
