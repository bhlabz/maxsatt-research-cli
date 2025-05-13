package delta

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/airbusgeo/godal"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/sentinel"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/utils"
	"github.com/schollz/progressbar/v3"
)

type Indexes struct {
	NDMI  []float64
	Cloud []float64
	SCL   []float64
	NDRE  []float64
	PSRI  []float64
	B02   []float64
	B04   []float64
	NDVI  []float64
}
type PixelData struct {
	Date      time.Time `csv:"date"`
	X         int       `csv:"x"`
	Y         int       `csv:"y"`
	Latitude  float64   `csv:"latitude"`
	Longitude float64   `csv:"longitude"`
	NDRE      float64   `csv:"ndre"`
	NDMI      float64   `csv:"ndmi"`
	PSRI      float64   `csv:"psri"`
	NDVI      float64   `csv:"ndvi"`
}

func xyToLatLon(dataset *godal.Dataset, x, y int) (float64, float64, error) {
	geoTransform, err := dataset.GeoTransform()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to retrieve GeoTransform: %w", err)
	}

	lat := geoTransform[1]*float64(x) + geoTransform[2]*float64(y) + geoTransform[1]*0.5 + geoTransform[2]*0.5 + geoTransform[0]
	lon := geoTransform[4]*float64(x) + geoTransform[5]*float64(y) + geoTransform[4]*0.5 + geoTransform[5]*0.5 + geoTransform[3]

	return lat, lon, nil
}

func CreatePixelDataset(farm, plot string, images map[time.Time]*godal.Dataset) ([]PixelData, error) {
	var width, height, totalPixels int

	for _, imageData := range images {
		width = imageData.Structure().SizeX
		height = imageData.Structure().SizeY
		totalPixels = width * height
		break
	}

	fileResults := []PixelData{}
	count := 0
	sortedImageDates := getSortedKeys(images)
	target := len(sortedImageDates) * width * height
	progressBar := progressbar.Default(int64(target), "Creating pixel dataset")

	var errGlobal error
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			for _, date := range sortedImageDates {
				image := images[date]
				result, err := getData(image, totalPixels, width, height, x, y, date)
				if err != nil {
					errGlobal = err
					break
				}

				if result != nil {
					result.Latitude, result.Longitude, err = xyToLatLon(image, x, y)
					if err != nil {
						errGlobal = err
						break
					}
					count++
					fileResults = append(fileResults, *result)
				}
				progressBar.Add(1)
			}
			if errGlobal != nil {
				break
			}
		}
		if errGlobal != nil {
			break
		}
	}
	progressBar.Finish()

	if errGlobal != nil {
		return nil, fmt.Errorf("error while creating pixel dataset: %w", errGlobal)
	}

	if len(fileResults) == 0 {
		return nil, fmt.Errorf("no data available to create the dataset for farm: %s, plot: %s using %d images from dates %v", farm, plot, len(images), sortedImageDates)
	}
	return fileResults, nil
}

func getData(image *godal.Dataset, totalPixels, width, height, x, y int, date time.Time) (*PixelData, error) {
	if totalPixels != 0 && totalPixels != width*height {
		return nil, errors.New("different image size")
	}
	var indexes map[string][][]float64
	var err error
	utils.ExecuteWithMutex(func() {
		indexes, err = sentinel.GetIndexesFromImage(image)
	})
	if err != nil {
		return nil, err
	}
	bands := sentinel.GetBands(indexes, x, y)

	if valid, _ := bands.Valid(); valid {
		return &PixelData{
			Date: date,
			X:    x,
			Y:    y,
			NDRE: bands.NDRE,
			NDMI: bands.NDMI,
			PSRI: bands.PSRI,
			NDVI: bands.NDVI,
		}, nil
	}
	return nil, nil
}

func getSortedKeys(m map[time.Time]*godal.Dataset) []time.Time {
	keys := make([]time.Time, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Before(keys[j])
	})
	return keys
}
