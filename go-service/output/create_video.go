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

const videoFPS = 2

func getDateFromPath(path string) (time.Time, error) {
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	parts := strings.Split(name, "_")

	// Try parsing format YYYY-MM-DD from the last part of the name
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		t, err := time.Parse("2006-01-02", lastPart)
		if err == nil {
			return t, nil
		}
	}

	// Try parsing format YYYY_MM_DD from the last 3 parts of the name
	if len(parts) >= 3 {
		dateStr := strings.Join(parts[len(parts)-3:], "_")
		t, err := time.Parse("2006_01_02", dateStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse date from filename: %s", name)
}

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

	// Sort image paths by date
	sort.Slice(imagePaths, func(i, j int) bool {
		dateI, errI := getDateFromPath(imagePaths[i])
		dateJ, errJ := getDateFromPath(imagePaths[j])
		if errI != nil || errJ != nil {
			// Fallback to string comparison if date parsing fails
			return imagePaths[i] < imagePaths[j]
		}
		return dateI.Before(dateJ)
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
	writer, err := mjpeg.New(outputPath, width, height, videoFPS)
	if err != nil {
		return err
	}
	defer writer.Close()

	fmt.Println("Video frame to date mapping:")
	// Encode and add each image
	for i, path := range imagePaths {
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

		date, err := getDateFromPath(path)
		if err == nil {
			frameTime := float64(i) / float64(videoFPS)
			fmt.Printf("  - %.2f seconds: %s (%s)\n", frameTime, date.Format("2006-01-02"), filepath.Base(path))
		} else {
			fmt.Printf("  - Could not extract date from: %s\n", filepath.Base(path))
		}
	}

	return nil
}
