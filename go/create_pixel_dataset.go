package main

import (
	"errors"
	"math"
	"sort"
	"time"

	"github.com/lukeroth/gdal"
	"github.com/schollz/progressbar/v3"
)

type Weather struct {
	Precipitation *float64
	Temperature   *float64
	Humidity      *float64
}

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

func createPixelDataset(images map[time.Time]gdal.Dataset, weather map[time.Time]Weather) ([]PixelData, error) {
	var width, height, totalPixels int
	var xRange, yRange []int

	for _, imageData := range images {
		width = imageData.RasterXSize()
		height = imageData.RasterYSize()
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
				result, err := getData(image, totalPixels, width, height, x, y, date, weather[date])
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

func getValues(indexes map[string][][]float64, x, y int) (float64, float64, float64, float64, float64, float64, float64, float64) {
	ndmiValue := indexes["ndmi"][y][x]
	cldValue := indexes["cloud"][y][x]
	sclValue := indexes["scl"][y][x]
	ndreValue := indexes["ndre"][y][x]
	psriValue := indexes["psri"][y][x]
	b02Value := indexes["b02"][y][x]
	b04Value := indexes["b04"][y][x]
	ndviValue := indexes["ndvi"][y][x]
	return ndmiValue, cldValue, sclValue, ndreValue, psriValue, b02Value, b04Value, ndviValue
}

func areIndexesValid(psriValue, ndviValue, ndmiValue, ndreValue, cldValue, sclValue, b02Value, b04Value float64) bool {
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

func isWeatherValid(weather Weather) bool {
	return weather.Precipitation == nil || weather.Temperature == nil
}

func getData(image gdal.Dataset, totalPixels, width, height, x, y int, date time.Time, weather Weather) (*PixelData, error) {
	if totalPixels != 0 && totalPixels != width*height {
		return nil, errors.New("different image size")
	}

	indexes := getIndexesFromImage(image)
	ndmiValue, cldValue, sclValue, ndreValue, psriValue, b02Value, b04Value, ndviValue := getValues(indexes, x, y)

	if isWeatherValid(weather) && areIndexesValid(psriValue, ndviValue, ndmiValue, ndreValue, cldValue, sclValue, b02Value, b04Value) {
		return &PixelData{
			Date:          date,
			X:             x,
			Y:             y,
			NDRE:          ndreValue,
			NDMI:          ndmiValue,
			PSRI:          psriValue,
			NDVI:          ndviValue,
			Temperature:   *weather.Temperature,
			Precipitation: *weather.Precipitation,
			Humidity:      *weather.Humidity,
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

func getSortedKeys(m map[time.Time]gdal.Dataset) []time.Time {
	keys := make([]time.Time, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Before(keys[j])
	})
	return keys
}
