package sentinel

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/airbusgeo/godal"
)

func getIndexesFromImage(ds *godal.Dataset) (map[string][][]float64, error) {
	bands := ds.Bands()
	bandsMap := make(map[string][][]float64)
	bandNames := []string{"B05", "B08", "B11", "B02", "B04", "B06", "CLD", "SCL"}
	for i, name := range bandNames {
		band := bands[i]
		width := ds.Structure().SizeX
		height := ds.Structure().SizeY
		data := make([][]float64, height)
		for y := 0; y < height; y++ {
			data[y] = make([]float64, width)
			err := band.Read(0, y, data[y], width, 1)
			if err != nil {
				return nil, fmt.Errorf("failed to read data for band %s: %w", name, err)
			}
		}
		bandsMap[name] = data
	}

	// Helper function for safe division
	safeDivide := func(a, b float64) float64 {
		if b == 0 {
			return 0
		}
		return a / b
	}

	// Calculate indexes
	height := len(bandsMap["B05"])
	width := len(bandsMap["B05"][0])
	indexes := map[string][][]float64{
		"ndre":  make([][]float64, height),
		"ndmi":  make([][]float64, height),
		"psri":  make([][]float64, height),
		"ndvi":  make([][]float64, height),
		"b02":   bandsMap["B02"],
		"b04":   bandsMap["B04"],
		"cloud": bandsMap["CLD"],
		"scl":   bandsMap["SCL"],
	}

	for y := 0; y < height; y++ {
		indexes["ndre"][y] = make([]float64, width)
		indexes["ndmi"][y] = make([]float64, width)
		indexes["psri"][y] = make([]float64, width)
		indexes["ndvi"][y] = make([]float64, width)
		for x := 0; x < width; x++ {
			b05 := bandsMap["B05"][y][x]
			b08 := bandsMap["B08"][y][x]
			b11 := bandsMap["B11"][y][x]
			b02 := bandsMap["B02"][y][x]
			b04 := bandsMap["B04"][y][x]
			b06 := bandsMap["B06"][y][x]

			indexes["ndre"][y][x] = safeDivide(b08-b05, b08+b05)
			indexes["ndmi"][y][x] = safeDivide(b08-b11, b08+b11)
			indexes["psri"][y][x] = safeDivide(b04-b02, b06)
			indexes["ndvi"][y][x] = safeDivide(b08-b04, b08+b04)
		}
	}

	return indexes, nil
}

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
func GetImages(geometry *godal.Geometry, farm, plot string, startDate, endDate time.Time, satelliteIntervalDays int) (map[time.Time]*godal.Dataset, error) {
	images := make(map[time.Time]*godal.Dataset)
	imagesNotFoundFile := "../data/images/invalid_images.json"

	// Load images_not_found.json
	var imagesNotFound []string
	if _, err := os.Stat(imagesNotFoundFile); err == nil {
		data, err := os.ReadFile(imagesNotFoundFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %v", imagesNotFoundFile, err)
		}
		if err := json.Unmarshal(data, &imagesNotFound); err != nil {
			return nil, fmt.Errorf("invalid JSON in %s: %v", imagesNotFoundFile, err)
		}
	}

	// Ensure images directory exists
	if _, err := os.Stat("../data/images"); os.IsNotExist(err) {
		if err := os.Mkdir("images", os.ModePerm); err != nil {
			return nil, fmt.Errorf("failed to create images directory: %v", err)
		}
	}

	// Iterate through dates
	for currentDate := startDate; !currentDate.After(endDate); currentDate = currentDate.AddDate(0, 0, satelliteIntervalDays) {
		startImageDate := currentDate
		endImageDate := currentDate.Add(time.Hour*23 + time.Minute*59 + time.Second*59)
		imageName := fmt.Sprintf("%s_%s_%s.tif", farm, plot, currentDate.Format("2006-01-02"))
		fileName := fmt.Sprintf("../data/images/%s_%s/%s", farm, plot, imageName)

		// Skip if image is in the not-found list
		if contains(imagesNotFound, imageName) {
			continue
		}

		// Skip if file already exists
		if _, err := os.Stat(fileName); err == nil {
			data, err := godal.Open(fileName, godal.ErrLogger(func(ec godal.ErrorCategory, code int, msg string) error {
				if ec == godal.CE_Warning {
					return nil
				}
				return err
			}))

			if err != nil {
				return nil, fmt.Errorf("failed to open %s: %v", fileName, err)
			}
			images[currentDate] = data
			continue
		}

		imageBytes, err := requestImage(startImageDate, endImageDate, geometry)
		if err != nil {
			if err.Error() == "Image not found" {
				imagesNotFound = append(imagesNotFound, fileName)
				saveImagesNotFound(imagesNotFoundFile, imagesNotFound)
				continue
			}
			return nil, fmt.Errorf("error requesting image: %v", err)
		}

		imagePath := filepath.Join("..", "data", "images", fmt.Sprintf("%s_%s", farm, plot))

		// Verifica se o diretório existe e cria caso não
		if _, err := os.Stat(imagePath); os.IsNotExist(err) {
			if mkErr := os.MkdirAll(imagePath, os.ModePerm); mkErr != nil {
				return nil, fmt.Errorf("failed to create directory %s: %v", imagePath, mkErr)
			}
		}

		permanentFileName := filepath.Join(imagePath, fmt.Sprintf("%s_%s_%s.tif", farm, plot, currentDate.Format("2006-01-02")))

		if err := os.WriteFile(permanentFileName, imageBytes, 0644); err != nil {
			return nil, fmt.Errorf("failed to write image file: %v", err)
		}

		ds, err := godal.Open(permanentFileName, godal.ErrLogger(func(ec godal.ErrorCategory, code int, msg string) error {
			if ec == godal.CE_Warning {
				return nil
			}
			return err
		}))
		if err != nil {
			fmt.Println(err.Error())
			return nil, err
		}

		indexes, err := getIndexesFromImage(ds)
		if err != nil {
			return nil, err
		}

		totalPixels := 100 // Placeholder for total pixels
		count := 0
		for y := 0; y < 10; y++ { // Placeholder for height
			for x := 0; x < 10; x++ { // Placeholder for width
				ndmiValue, cldValue, sclValue, ndreValue, psriValue, b02Value, b04Value, ndviValue := GetValues(indexes, x, y)
				if !AreIndexesValid(ndmiValue, cldValue, sclValue, ndreValue, psriValue, b02Value, b04Value, ndviValue) {
					count++
				}
			}
		}
		if count == totalPixels {
			imagesNotFound = append(imagesNotFound, imageName)
			saveImagesNotFound(imagesNotFoundFile, imagesNotFound)
			if err := os.Remove(permanentFileName); err != nil {
				fmt.Printf("failed to delete image file %s: %v\n", permanentFileName, err)
			}
			continue
		}

		images[currentDate] = ds
	}

	return images, nil
}

func saveImagesNotFound(filePath string, imagesNotFound []string) {
	var existingImagesNotFound []string

	// Check if the file exists
	if _, err := os.Stat(filePath); err == nil {
		// File exists, read and unmarshal its content
		data, err := os.ReadFile(filePath)
		if err == nil {
			_ = json.Unmarshal(data, &existingImagesNotFound)
		}
	}

	// Append new images to the existing list
	existingImagesNotFound = append(existingImagesNotFound, imagesNotFound...)

	// Remove duplicates
	uniqueImages := make(map[string]struct{})
	for _, image := range existingImagesNotFound {
		uniqueImages[image] = struct{}{}
	}

	// Convert back to a slice
	finalImagesNotFound := make([]string, 0, len(uniqueImages))
	for image := range uniqueImages {
		finalImagesNotFound = append(finalImagesNotFound, image)
	}

	// Marshal and write back to the file
	data, _ := json.Marshal(finalImagesNotFound)
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
