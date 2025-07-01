package output

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/icza/mjpeg"
)

// CreateVideoFromDirectory creates a video from all images in a directory, sorted by filename
func CreateVideoFromDirectory(dirPath string, outputPath string) error {
	// Find all image files in the directory
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return err
	}

	var imagePaths []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
			imagePaths = append(imagePaths, filepath.Join(dirPath, file.Name()))
		}
	}

	if len(imagePaths) == 0 {
		return fmt.Errorf("no image files found in directory: %s", dirPath)
	}

	// Sort image paths by date at the end of the filename (format: ..._YYYY_MM_DD)
	sort.Slice(imagePaths, func(i, j int) bool {
		getDate := func(path string) time.Time {
			base := filepath.Base(path)
			dot := strings.LastIndex(base, ".")
			if dot == -1 {
				return time.Time{}
			}
			name := base[:dot]
			parts := strings.Split(name, "_")
			if len(parts) < 3 {
				return time.Time{}
			}
			dateStr := strings.Join(parts[len(parts)-3:], "_")
			t, err := time.Parse("2006_01_02", dateStr)
			if err != nil {
				return time.Time{}
			}
			return t
		}
		return getDate(imagePaths[i]).Before(getDate(imagePaths[j]))
	})

	// Determine the maximum width and height from all images
	var maxWidth, maxHeight int32
	for _, path := range imagePaths {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			return err
		}
		bounds := img.Bounds()
		width := int32(bounds.Dx())
		height := int32(bounds.Dy())
		if width > maxWidth {
			maxWidth = width
		}
		if height > maxHeight {
			maxHeight = height
		}
	}
	width := maxWidth
	height := maxHeight

	// Create video writer
	if !strings.Contains(outputPath, ".avi") {
		outputPath += ".avi"
	}
	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return err
	}
	writer, err := mjpeg.New(outputPath, width, height, 2)
	if err != nil {
		return err
	}
	defer writer.Close()

	// Encode and add each image
	for _, path := range imagePaths {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			return err
		}

		var buf bytes.Buffer
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 100})
		if err != nil {
			return err
		}

		err = writer.AddFrame(buf.Bytes())
		if err != nil {
			return err
		}
	}

	return nil
}
