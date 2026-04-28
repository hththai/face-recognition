package main

import (
	"fmt"
	"os"

	"myproject/service"
)

func main() {
	if err := service.ConvertFolder("original-images", "convert-folder"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
