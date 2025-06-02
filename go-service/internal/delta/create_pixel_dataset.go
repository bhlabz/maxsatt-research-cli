package delta

import (
	"errors"
	"fmt"
	"image/color"
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
	Date                 time.Time `csv:"date"`
	X                    int       `csv:"x"`
	Y                    int       `csv:"y"`
	Latitude             float64   `csv:"latitude"`
	Longitude            float64   `csv:"longitude"`
	NDRE                 float64   `csv:"ndre"`
	NDMI                 float64   `csv:"ndmi"`
	PSRI                 float64   `csv:"psri"`
	NDVI                 float64   `csv:"ndvi"`
	Status               sentinel.PixelStatus
	Color                *color.RGBA
	mostRecentValidPixel *PixelData `csv:"-"`
}

func xyToLatLon(dataset *godal.Dataset, x, y int) (float64, float64, error) {
	geoTransform, err := dataset.GeoTransform()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get GeoTransform: %w", err)
	}

	xCoord := geoTransform[0] + geoTransform[1]*(float64(x)+0.5) + geoTransform[2]*(float64(y)+0.5)
	yCoord := geoTransform[3] + geoTransform[4]*(float64(x)+0.5) + geoTransform[5]*(float64(y)+0.5)

	// Transform to WGS84
	srcSR := dataset.SpatialRef()
	defer srcSR.Close()
	dstSR, _ := godal.NewSpatialRefFromEPSG(4326) // WGS84
	defer dstSR.Close()
	tr, _ := godal.NewTransform(srcSR, dstSR)
	defer tr.Close()

	xs := []float64{xCoord}
	ys := []float64{yCoord}
	if err := tr.TransformEx(xs, ys, nil, nil); err != nil {
		return 0, 0, fmt.Errorf("transform error: %w", err)
	}

	return ys[0], xs[0], nil
}

func CreatePixelDataset(farm, plot string, images map[time.Time]*godal.Dataset) (map[[2]int][]PixelData, error) {
	var width, height, totalPixels int

	for _, imageData := range images {
		width = imageData.Structure().SizeX
		height = imageData.Structure().SizeY
		totalPixels = width * height
		break
	}

	historicalPixelDataset := make(map[[2]int][]PixelData)
	sortedImageDates := getSortedKeys(images)
	target := len(sortedImageDates) * width * height
	progressBar := progressbar.Default(int64(target), "Creating pixel dataset")

	var errGlobal error
	for _, date := range sortedImageDates {
		pixelDataset := make(map[[2]int]PixelData)
		treatablePixelsCount, invalidPixelsCount, validPixelsCount := 0, 0, 0
		image := images[date]
		for y := range height {
			for x := range width {
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
					pixelDataset[[2]int{x, y}] = *result

					switch result.Status {
					case sentinel.PixelStatusTreatable:
						treatablePixelsCount++
					case sentinel.PixelStatusInvalid:
						invalidPixelsCount++
					case sentinel.PixelStatusValid:
						validPixelsCount++
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
		if errGlobal != nil {
			continue
		}

		if validPixelsCount == 0 {
			continue
		}

		for k, pixelData := range pixelDataset {
			historicalPixelDataset[k] = append(historicalPixelDataset[k], pixelData)
		}
	}

	progressBar.Finish()

	if errGlobal != nil {
		return nil, fmt.Errorf("error while creating pixel dataset: %w", errGlobal)
	}

	if len(historicalPixelDataset) == 0 {
		return nil, fmt.Errorf("no data available to create the dataset for farm: %s, plot: %s using %d images from dates %v", farm, plot, len(images), sortedImageDates)
	}
	return historicalPixelDataset, nil
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

	pixelStatus := bands.Valid()
	return &PixelData{
		Date:   date,
		X:      x,
		Y:      y,
		NDRE:   bands.NDRE,
		NDMI:   bands.NDMI,
		PSRI:   bands.PSRI,
		NDVI:   bands.NDVI,
		Status: pixelStatus,
	}, nil

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
