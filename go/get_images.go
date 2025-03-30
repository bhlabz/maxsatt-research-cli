package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/lukeroth/gdal"
)

// GetImages retrieves satellite images based on the given parameters
func GetImages(geometry map[string]any, farm, plot string, startDate, endDate time.Time, satelliteIntervalDays int) (map[string]gdal.Dataset, error) {
	images := make(map[string]gdal.Dataset)
	imagesNotFoundFile := "images/images_not_found.json"

	// Load images_not_found.json
	var imagesNotFound []string
	if _, err := os.Stat(imagesNotFoundFile); err == nil {
		data, err := ioutil.ReadFile(imagesNotFoundFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %v", imagesNotFoundFile, err)
		}
		if err := json.Unmarshal(data, &imagesNotFound); err != nil {
			return nil, fmt.Errorf("invalid JSON in %s: %v", imagesNotFoundFile, err)
		}
	}

	// Ensure images directory exists
	if _, err := os.Stat("images"); os.IsNotExist(err) {
		if err := os.Mkdir("images", os.ModePerm); err != nil {
			return nil, fmt.Errorf("failed to create images directory: %v", err)
		}
	}

	// Iterate through dates
	for currentDate := startDate; !currentDate.After(endDate); currentDate = currentDate.AddDate(0, 0, satelliteIntervalDays) {
		startDateStr := currentDate.Format("2006-01-02T00:00:00Z")
		endDateStr := currentDate.Format("2006-01-02T23:59:59Z")
		fileName := fmt.Sprintf("images/%s_%s/%s_%s.tif", farm, plot, farm, currentDate.Format("2006-01-02"))

		// Skip if image is in the not-found list
		if contains(imagesNotFound, fileName) {
			continue
		}

		// Skip if file already exists
		if _, err := os.Stat(fileName); err == nil {
			data, err := gdal.Open(fileName, gdal.Access(gdal.ReadOnly))
			if err != nil {
				return nil, fmt.Errorf("failed to open %s: %v", fileName, err)
			}
			images[currentDate.Format("2006-01-02")] = data
			continue
		}

		// Request image
		image, err := requestImage(startDateStr, endDateStr, geometry, 0, 0) // Width and height are placeholders
		if err != nil {
			if err.Error() == "Image not found" {
				imagesNotFound = append(imagesNotFound, fileName)
				saveImagesNotFound(imagesNotFoundFile, imagesNotFound)
				continue
			}
			return nil, fmt.Errorf("error requesting image: %v", err)
		}

		// Validate image (mocked logic)
		totalPixels := 100 // Placeholder for total pixels
		count := 0
		for y := 0; y < 10; y++ { // Placeholder for height
			for x := 0; x < 10; x++ { // Placeholder for width
				_, _, _, _, _, _, _, _ = getValues(nil, x, y)
				if !areIndexesValid(0, 0, 0, 0, 0, 0, 0, 0) {
					count++
				}
			}
		}
		if count == totalPixels {
			imagesNotFound = append(imagesNotFound, fileName)
			saveImagesNotFound(imagesNotFoundFile, imagesNotFound)
			continue
		}

		// Save image to file
		dirPath := filepath.Dir(fileName)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
				return nil, fmt.Errorf("failed to create directory %s: %v", dirPath, err)
			}
		}
		if err := ioutil.WriteFile(fileName, image, 0644); err != nil {
			return nil, fmt.Errorf("failed to save image to %s: %v", fileName, err)
		}
		data, err := gdal.Open(fileName, gdal.Access(gdal.ReadOnly))
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %v", fileName, err)
		}

		images[currentDate.Format("2006-01-02")] = data
	}

	return images, nil
}

func saveImagesNotFound(filePath string, imagesNotFound []string) {
	data, _ := json.Marshal(imagesNotFound)
	_ = os.WriteFile(filePath, data, 0644)
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// func main() {
// 	// Example usage
// 	geometry := Geometry{Coordinates: nil, CRS: "WGS84"}
// 	startDate, _ := time.Parse("2006-01-02", "2023-01-01")
// 	endDate, _ := time.Parse("2006-01-02", "2023-01-10")
// 	images, err := GetImages(geometry, "farm1", "plot1", startDate, endDate, 5)
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 		return
// 	}
// 	fmt.Println("Images retrieved:", len(images))
// }
