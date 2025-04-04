package delta

import (
	"errors"
	"sort"
	"time"

	"github.com/airbusgeo/godal"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/sentinel"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/weather"

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
	Farm          string
	Plot          string
	Date          time.Time
	X             int
	Y             int
	NDRE          float64
	NDMI          float64
	PSRI          float64
	NDVI          float64
	Precipitation float64
	Temperature   float64
	Humidity      float64
}

func createPixelDataset(farm, plot string, images map[time.Time]*godal.Dataset, weather map[time.Time]weather.Weather) ([]PixelData, error) {
	var width, height, totalPixels int
	var xRange, yRange []int

	for _, imageData := range images {
		width = imageData.Structure().SizeX
		height = imageData.Structure().SizeY
		xRange = makeRange(0, width)
		yRange = makeRange(0, height)
		totalPixels = width * height
		break
	}

	fileResults := []PixelData{}
	count := 0
	target := len(yRange) * len(xRange) * len(images)
	progressBar := progressbar.Default(int64(target))

	for _, y := range yRange {
		for _, x := range xRange {
			sortedImageDates := getSortedKeys(images)
			for _, date := range sortedImageDates {
				image := images[date]
				result, err := getData(farm, plot, image, totalPixels, width, height, x, y, date, weather[date])
				if err != nil {
					return nil, err
				}
				count++
				if result != nil {
					fileResults = append(fileResults, *result)
				}
				progressBar.Add(1)
			}
		}
	}

	if len(fileResults) == 0 {
		return nil, errors.New("no data available to create the dataset")
	}
	return fileResults, nil
}

func getData(farm, plot string, image *godal.Dataset, totalPixels, width, height, x, y int, date time.Time, weather weather.Weather) (*PixelData, error) {
	if totalPixels != 0 && totalPixels != width*height {
		return nil, errors.New("different image size")
	}

	indexes, err := sentinel.GetIndexesFromImage(image)
	if err != nil {
		return nil, err
	}

	ndmiValue, cldValue, sclValue, ndreValue, psriValue, b02Value, b04Value, ndviValue := sentinel.GetValues(indexes, x, y)

	if sentinel.AreIndexesValid(psriValue, ndviValue, ndmiValue, ndreValue, cldValue, sclValue, b02Value, b04Value) {
		return &PixelData{
			Farm:          farm,
			Plot:          plot,
			Date:          date,
			X:             x,
			Y:             y,
			NDRE:          ndreValue,
			NDMI:          ndmiValue,
			PSRI:          psriValue,
			NDVI:          ndviValue,
			Temperature:   weather.Temperature,
			Precipitation: weather.Precipitation,
			Humidity:      weather.Humidity,
		}, nil
	}
	return nil, nil
}

func makeRange(min, max int) []int {
	r := make([]int, max-min)
	for i := range r {
		r[i] = min + i
	}
	return r
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
