package sentinel

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/airbusgeo/godal"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
	"github.com/schollz/progressbar/v3"
)

type Bands struct {
	NDMI, CLD, SCL, NDRE, PSRI, B02, B04, NDVI float64
}
type PixelStatus string

var (
	PixelStatusValid     PixelStatus = "valid"
	PixelStatusInvalid   PixelStatus = "invalid"
	PixelStatusUnknown   PixelStatus = "unknown"
	PixelStatusTreatable PixelStatus = "treatable"
)

func reprojectAutoUTM(inputPath, outputPath string) error {
	// Register drivers and open source GeoTIFF
	godal.RegisterAll()
	ds, err := godal.Open(inputPath, godal.ErrLogger(func(ec godal.ErrorCategory, code int, msg string) error {
		if ec == godal.CE_Warning {
			return nil
		}
		return fmt.Errorf("error opening dataset: %s", msg)
	}))
	if err != nil {
		return err
	}
	defer ds.Close()

	// Inspect current projection
	sr := ds.SpatialRef()
	defer sr.Close()

	// Compute image center in source CRS
	bounds, err := ds.Bounds()
	if err != nil {
		return err
	}
	centerX := (bounds[0] + bounds[2]) / 2.0
	centerY := (bounds[1] + bounds[3]) / 2.0

	// Transform to WGS84 lat/lon if needed
	var lon, lat float64
	if !sr.EPSGTreatsAsLatLong() {
		dstSR, _ := godal.NewSpatialRefFromEPSG(4326)
		defer dstSR.Close()
		tr, _ := godal.NewTransform(sr, dstSR)
		defer tr.Close()
		xs := []float64{centerX}
		ys := []float64{centerY}
		err := tr.TransformEx(xs, ys, nil, nil)
		if err != nil {
			panic(err)
		}
		lon = xs[0]
		lat = ys[0]
	} else {
		lon, lat = centerX, centerY
	}

	// Compute UTM zone/EPSG
	zone := int(math.Floor((lon+180.0)/6.0)) + 1
	var utmEPSG int
	if lat >= 0 {
		utmEPSG = 32600 + zone
	} else {
		utmEPSG = 32700 + zone
	}
	utmCode := fmt.Sprintf("EPSG:%d", utmEPSG)
	// Reproject (warp) to UTM and save result
	outDS, err := ds.Warp(outputPath,
		[]string{"-t_srs", utmCode},
		godal.CreationOption("TILED=YES", "COMPRESS=LZW"),
		godal.GTiff, godal.ErrLogger(func(ec godal.ErrorCategory, code int, msg string) error {
			if ec == godal.CE_Warning {
				return nil
			}
			return fmt.Errorf("error warp dataset: %s", msg)
		}))
	if err != nil {
		return err
	}
	defer outDS.Close()

	return nil
}

func GetBands(indexes map[string][][]float64, x, y int) Bands {
	ndmiValue := indexes["ndmi"][y][x]
	cldValue := indexes["cloud"][y][x]
	sclValue := indexes["scl"][y][x]
	ndreValue := indexes["ndre"][y][x]
	psriValue := indexes["psri"][y][x]
	b02Value := indexes["b02"][y][x]
	b04Value := indexes["b04"][y][x]
	ndviValue := indexes["ndvi"][y][x]
	return Bands{
		NDMI: ndmiValue,
		CLD:  cldValue,
		SCL:  sclValue,
		NDRE: ndreValue,
		PSRI: psriValue,
		B02:  b02Value,
		B04:  b04Value,
		NDVI: ndviValue,
	}
}

func (bands Bands) Valid() PixelStatus {
	invalidConditions := []struct {
		Condition   bool
		PixelStatus PixelStatus
	}{
		{bands.CLD > 0, PixelStatusUnknown},
		{bands.SCL == 2 || bands.SCL == 3 || bands.SCL == 10, PixelStatusUnknown},
		{bands.SCL == 8 || bands.SCL == 9, PixelStatusUnknown},
		{(bands.B04+bands.B02)/2 > 0.9, PixelStatusInvalid},
		{bands.PSRI == 0 && bands.NDVI == 0 && bands.NDMI == 0 && bands.NDRE == 0, PixelStatusInvalid},
	}

	for _, condition := range invalidConditions {
		if condition.Condition {
			return condition.PixelStatus
		}
	}
	return PixelStatusValid
}

// GetImages retrieves satellite images based on the given parameters
func GetImages(geometry *godal.Geometry, forest, plot string, startDate, endDate time.Time, satelliteIntervalDays int) (map[time.Time]*godal.Dataset, error) {
	images := make(map[time.Time]*godal.Dataset)
	imagesNotFoundFile := fmt.Sprintf("%s/data/images/invalid_images.json", properties.RootPath())

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
	if _, err := os.Stat(fmt.Sprintf("%s/data/images", properties.RootPath())); os.IsNotExist(err) {
		if err := os.Mkdir("images", os.ModePerm); err != nil {
			return nil, fmt.Errorf("failed to create images directory: %v", err)
		}
	}

	// Iterate through dates
	progressbar := progressbar.Default(int64(endDate.Sub(startDate).Hours()/24), "Getting images")
	for currentDate := startDate; !currentDate.After(endDate); currentDate = currentDate.AddDate(0, 0, satelliteIntervalDays) {
		startImageDate := currentDate
		endImageDate := currentDate.Add(time.Hour*23 + time.Minute*59 + time.Second*59)
		imageName := fmt.Sprintf("%s_%s_%s.tif", forest, plot, currentDate.Format("2006-01-02"))
		fileName := fmt.Sprintf("%s/data/images/%s_%s/%s", properties.RootPath(), forest, plot, imageName)

		// Skip if image is in the not-found list
		if contains(imagesNotFound, imageName) {
			progressbar.Add(1)
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
			progressbar.Add(1)
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

		imagePath := fmt.Sprintf("%s/data/images/%s_%s", properties.RootPath(), forest, plot)
		// Verifica se o diretório existe e cria caso não
		if _, err := os.Stat(imagePath); os.IsNotExist(err) {
			if mkErr := os.MkdirAll(imagePath, os.ModePerm); mkErr != nil {
				return nil, fmt.Errorf("failed to create directory %s: %v", imagePath, mkErr)
			}
		}

		permImageName := filepath.Join(imagePath, imageName)
		tempImageName := imagePath + ".temp"
		if err := os.WriteFile(tempImageName, imageBytes, 0644); err != nil {
			return nil, fmt.Errorf("failed to write image file: %v", err)
		}
		defer os.Remove(tempImageName)

		if err = reprojectAutoUTM(tempImageName, permImageName); err != nil {
			return nil, fmt.Errorf("failed to reproject image: %v", err)
		}

		ds, err := godal.Open(permImageName, godal.ErrLogger(func(ec godal.ErrorCategory, code int, msg string) error {
			if ec == godal.CE_Warning {
				return nil
			}
			return err
		}))
		if err != nil {
			fmt.Println(err.Error())
			return nil, err
		}

		indexes, err := GetIndexesFromImage(ds)
		if err != nil {
			return nil, err
		}

		height := len(indexes["ndmi"])
		width := len(indexes["ndmi"][0])
		totalPixels := height * width
		count := 0
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				bands := GetBands(indexes, x, y)
				pixelStatus := bands.Valid()
				if pixelStatus == PixelStatusInvalid {
					count++
				}
			}
		}
		if count == totalPixels {
			imagesNotFound = append(imagesNotFound, imageName)
			saveImagesNotFound(imagesNotFoundFile, imagesNotFound)
			if err := os.Remove(permImageName); err != nil {
				fmt.Printf("failed to delete image file %s: %v\n", permImageName, err)
			}
			progressbar.Add(1)
			continue
		}

		images[currentDate] = ds
		progressbar.Add(1)
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
// 	images, err := GetImages(geometry, "forest1", "plot1", startDate, endDate, 5)
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 		return
// 	}
// 	fmt.Println("Images retrieved:", len(images))
// }
