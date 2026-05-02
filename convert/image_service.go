package convertservice

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/disintegration/imaging"
	exif "github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
	_ "golang.org/x/image/webp"
)

const MaxFileSizeMB = 5

func LoadImageWithOrientation(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	rawExif, err := exif.SearchFileAndExtractExif(path)
	if err != nil {
		return img, nil
	}

	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		return img, nil
	}
	ti := exif.NewTagIndex()
	_, index, err := exif.Collect(im, ti, rawExif)
	if err != nil {
		return img, nil
	}

	tags, err := index.RootIfd.FindTagWithName("Orientation")
	if err != nil || len(tags) == 0 {
		return img, nil
	}

	value, err := tags[0].Value()
	if err != nil {
		return img, nil
	}

	orientations, ok := value.([]uint16)
	if !ok || len(orientations) == 0 {
		return img, nil
	}

	switch orientations[0] {
	case 3:
		img = imaging.Rotate180(img)
	case 6:
		img = imaging.Rotate270(img)
	case 8:
		img = imaging.Rotate90(img)
	}

	return img, nil
}

func CompressToMaxSize(img image.Image, dstPath string, maxMB int) error {
	img = imaging.Resize(img, 2000, 0, imaging.Lanczos)

	lo, hi := 10, 90
	var bestBuf *bytes.Buffer

	for lo <= hi {
		mid := (lo + hi) / 2
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: mid}); err != nil {
			return fmt.Errorf("encode: %w", err)
		}
		if buf.Len() <= maxMB*1024*1024 {
			bestBuf = &buf
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}

	if bestBuf == nil {
		return fmt.Errorf("cannot compress below %dMB", maxMB)
	}
	return os.WriteFile(dstPath, bestBuf.Bytes(), 0644)
}

func ResizeImage(srcPath, dstPath string) error {
	img, err := LoadImageWithOrientation(srcPath)
	if err != nil {
		return err
	}
	return CompressToMaxSize(img, dstPath, MaxFileSizeMB)
}

func ConvertFolder(srcDir, dstDir string) error {
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("create dst dir: %w", err)
	}

	var paths []string
	filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		ext := filepath.Ext(info.Name())
		if ext == ".JPG" || ext == ".jpg" || ext == ".webp" || ext == ".png" {
			paths = append(paths, path)
		}
		return nil
	})

	totalStart := time.Now()
	sem := make(chan struct{}, runtime.NumCPU())
	var wg sync.WaitGroup

	for _, p := range paths {
		wg.Add(1)
		sem <- struct{}{}
		go func(srcPath string) {
			defer wg.Done()
			defer func() { <-sem }()
			dstPath := filepath.Join(dstDir, filepath.Base(srcPath))
			start := time.Now()
			if err := ResizeImage(srcPath, dstPath); err != nil {
				fmt.Printf("❌ Failed: %s → %v\n", srcPath, err)
			} else {
				fmt.Printf("✅ Converted: %s → %s (%.2fs)\n", srcPath, dstPath, time.Since(start).Seconds())
			}
		}(p)
	}

	wg.Wait()
	fmt.Printf("Total time: %.2fs\n", time.Since(totalStart).Seconds())
	return nil
}
