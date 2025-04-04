package sentinel

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/airbusgeo/godal"
)

func GetValues(indexes map[string][][]float64, x, y int) (ndmiValue, cldValue, sclValue, ndreValue, psriValue, b02Value, b04Value, ndviValue float64) {
	ndmiValue = indexes["ndmi"][y][x]
	cldValue = indexes["cloud"][y][x]
	sclValue = indexes["scl"][y][x]
	ndreValue = indexes["ndre"][y][x]
	psriValue = indexes["psri"][y][x]
	b02Value = indexes["b02"][y][x]
	b04Value = indexes["b04"][y][x]
	ndviValue = indexes["ndvi"][y][x]
	return ndmiValue, cldValue, sclValue, ndreValue, psriValue, b02Value, b04Value, ndviValue
}

func AreIndexesValid(ndmiValue, cldValue, sclValue, ndreValue, psriValue, b02Value, b04Value, ndviValue float64) bool {
	invalidConditions := []struct {
		Condition bool
		Reason    string
	}{
		{math.IsNaN(psriValue), "PSRI value is NaN"},
		{math.IsNaN(ndviValue), "NDVI value is NaN"},
		{math.IsNaN(ndmiValue), "NDMI value is NaN"},
		{math.IsNaN(ndreValue), "NDRE value is NaN"},
		{cldValue > 0, "Cloud value is greater than 0"},
		{sclValue == 3 || sclValue == 8 || sclValue == 9 || sclValue == 10, "SCL value is in [3, 8, 9, 10]"},
		{(b04Value+b02Value)/2 > 0.9, "(B04 value + B02 value) / 2 is greater than 0.9"},
		{psriValue == 0 && ndviValue == 0 && ndmiValue == 0 && ndreValue == 0, "All index values are 0"},
	}

	for _, condition := range invalidConditions {
		if condition.Condition {
			return false
		}
	}
	return true
}

// GetImages retrieves satellite images based on the given parameters
func GetImages(geometry map[string]any, farm, plot string, startDate, endDate time.Time, satelliteIntervalDays int) (map[time.Time]*godal.Dataset, error) {
	images := make(map[time.Time]*godal.Dataset)
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
			data, err := godal.Open(fileName)
			if err != nil {
				return nil, fmt.Errorf("failed to open %s: %v", fileName, err)
			}
			images[currentDate] = data
			continue
		}

		// Request image TODO: image width and height should be calculated based on geometry
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
				ndmiValue, cldValue, sclValue, ndreValue, psriValue, b02Value, b04Value, ndviValue := GetValues(nil, x, y)
				if !AreIndexesValid(ndmiValue, cldValue, sclValue, ndreValue, psriValue, b02Value, b04Value, ndviValue) {
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
		data, err := godal.Open(fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %v", fileName, err)
		}

		images[currentDate] = data
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
