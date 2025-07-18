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
func GetImages(geometry *godal.Geometry, farm, plot string, startDate, endDate time.Time) (map[time.Time]*godal.Dataset, error) {
	images := make(map[time.Time]*godal.Dataset)

	// Ensure images directory exists
	if _, err := os.Stat(fmt.Sprintf("%s/data/images", properties.RootPath())); os.IsNotExist(err) {
		if err := os.MkdirAll(fmt.Sprintf("%s/data/images", properties.RootPath()), os.ModePerm); err != nil {
			return nil, fmt.Errorf("failed to create images directory: %v", err)
		}
	}

	bandData, err := requestImage(startDate, endDate, []*godal.Geometry{geometry})
	if err != nil {
		return nil, fmt.Errorf("error requesting image: %v", err)
	}

	imagePath := fmt.Sprintf("%s/data/images/%s_%s", properties.RootPath(), farm, plot)
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		if mkErr := os.MkdirAll(imagePath, os.ModePerm); mkErr != nil {
			return nil, fmt.Errorf("failed to create directory %s: %v", imagePath, mkErr)
		}
	}

	// Find all unique dates
	dateSet := make(map[time.Time]struct{})
	for _, dateMap := range bandData {
		for date := range dateMap[0] {
			dateSet[date] = struct{}{}
		}
	}

	progressbar := progressbar.Default(int64(len(dateSet)), "Creating temp. tiff image")

	for date := range dateSet {
		var mainBandFile string
		for band, dateMap := range bandData {
			data, ok := dateMap[0][date]
			if !ok {
				continue
			}
			bandFile := filepath.Join(imagePath, fmt.Sprintf("%s_%s_%s_%s.tif", farm, plot, date.Format("2006-01-02"), band))
			if err := os.WriteFile(bandFile, data, 0644); err != nil {
				return nil, fmt.Errorf("failed to write band file %s: %v", bandFile, err)
			}
			if band == "B04" || band == "B08" {
				mainBandFile = bandFile
			}
		}
		if mainBandFile == "" {
			// fallback to any band for this date
			for band, dateMap := range bandData {
				_, ok := dateMap[0][date]
				if ok {
					mainBandFile = filepath.Join(imagePath, fmt.Sprintf("%s_%s_%s_%s.tif", farm, plot, date.Format("2006-01-02"), band))
					break
				}
			}
		}
		if mainBandFile == "" {
			progressbar.Add(1)
			continue
		}
		ds, err := godal.Open(mainBandFile, godal.ErrLogger(func(ec godal.ErrorCategory, code int, msg string) error {
			if ec == godal.CE_Warning {
				return nil
			}
			return err
		}))
		if err != nil {
			fmt.Println(err.Error())
			progressbar.Add(1)
			continue
		}
		images[date] = ds
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
// 	images, err := GetImages(geometry, "farm1", "plot1", startDate, endDate, 5)
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 		return
// 	}
// 	fmt.Println("Images retrieved:", len(images))
// }
