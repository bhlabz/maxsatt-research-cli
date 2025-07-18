package delivery

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/dataset"
	delta1 "github.com/forest-guardian/forest-guardian-api-poc/internal/dataset"
	ml "github.com/forest-guardian/forest-guardian-api-poc/internal/ml"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/sentinel"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/weather"
)

func EvaluatePlotCleanData(farm, plot string, endDate time.Time) ([]dataset.PixelData, error) {
	startDate := endDate.AddDate(0, 0, -40)

	geometry, err := sentinel.GetGeometryFromGeoJSON(farm, plot)
	if err != nil {
		return nil, err
	}

	images, err := sentinel.GetImages(geometry, farm, plot, startDate, endDate, 1)
	if err != nil {
		return nil, err
	}

	data, err := dataset.CreatePixelDataset(farm, plot, images)
	if err != nil {
		return nil, err
	}

	cleanDataset, err := dataset.CreateCleanDataset(farm, plot, data)
	if err != nil {
		return nil, err
	}

	groupedData := make(map[time.Time][]dataset.PixelData)
	for _, sortedPixels := range cleanDataset {
		for date, pixel := range sortedPixels {
			//if pixel.Status == sentinel.PixelStatusTreatable {
			//	fmt.Println("TREATABLE FOUND")
			//}
			groupedData[date] = append(groupedData[date], pixel)
		}
	}

	var mostRecentDate time.Time
	for date := range groupedData {
		if date.After(mostRecentDate) {
			mostRecentDate = date
		}
	}

	return groupedData[mostRecentDate], nil
}

func EvaluatePlotDeltaData(deltaDays, deltaDaysThreshold int, farm, plot string, endDate time.Time) (map[[2]int]map[time.Time]dataset.DeltaData, error) {

	getDaysBeforeEvidenceToAnalyse := deltaDays + deltaDaysThreshold
	startDate := endDate.AddDate(0, 0, -getDaysBeforeEvidenceToAnalyse)

	geometry, err := sentinel.GetGeometryFromGeoJSON(farm, plot)
	if err != nil {
		return nil, err
	}

	images, err := sentinel.GetImages(geometry, farm, plot, startDate, endDate, 1)
	if err != nil {
		return nil, err
	}

	data, err := dataset.CreatePixelDataset(farm, plot, images)
	if err != nil {
		return nil, err
	}

	cleanData, err := dataset.CreateCleanDataset(farm, plot, data)
	if err != nil {
		return nil, err
	}

	deltaDataset, err := dataset.CreateDeltaDataset(farm, plot, deltaDays, deltaDaysThreshold, cleanData)
	if err != nil {
		return nil, err
	}

	return deltaDataset, nil
}

func EvaluatePlotFinalData(model, farm, plot string, endDate time.Time) ([]ml.PixelResult, error) {
	start := time.Now()
	// Parse the model string according to the expected format
	fmt.Println("Parsing model string:", model)
	model = strings.TrimSuffix(model, ".csv")
	parts := strings.Split(model, "_")
	if len(parts) != 8 {
		return nil, fmt.Errorf("model string has %d parts, expected 8: %v", len(parts), parts)
	}
	_, err := strconv.Atoi(parts[0]) // modelID, unused
	if err != nil {
		return nil, err
	}
	// modelDate := parts[1], unused
	deltaDays, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, err
	}
	deltaDaysThreshold, err := strconv.Atoi(parts[3])
	if err != nil {
		return nil, err
	}
	daysBeforeEvidenceToAnalyze, err := strconv.Atoi(parts[4])
	if err != nil {
		return nil, err
	}
	if parts[5] != "training" {
		return nil, fmt.Errorf("expected 'training' literal in model string, got '%s'", parts[5])
	}
	_, err = strconv.Atoi(parts[7]) // trainingRatio, unused
	if err != nil {
		return nil, err
	}

	daysBeforeEvidenceToFetch := deltaDays + deltaDaysThreshold + daysBeforeEvidenceToAnalyze

	endDate = endDate.AddDate(0, 0, -daysBeforeEvidenceToAnalyze)
	startDate := endDate.AddDate(0, 0, -daysBeforeEvidenceToFetch)

	fmt.Println("daysBeforeEvidenceToAnalyze", daysBeforeEvidenceToAnalyze)
	fmt.Println("daysBeforeEvidenceToFetch", daysBeforeEvidenceToFetch)
	fmt.Println("deltaDays", deltaDays)
	fmt.Println("deltaDaysThreshold", deltaDaysThreshold)
	fmt.Println("startDate", startDate)
	fmt.Println("endDate", endDate)

	geometry, err := sentinel.GetGeometryFromGeoJSON(farm, plot)
	if err != nil {
		return nil, err
	}

	stepStart := time.Now()
	images, err := sentinel.GetImages(geometry, farm, plot, startDate, endDate, 1)
	if err != nil {
		return nil, err
	}
	fmt.Printf("GetImages took %v\n", time.Since(stepStart))

	stepStart = time.Now()
	data, err := dataset.CreatePixelDataset(farm, plot, images)
	if err != nil {
		return nil, err
	}

	cleanData, err := dataset.CreateCleanDataset(farm, plot, data)
	if err != nil {
		return nil, err
	}

	deltaDataset, err := dataset.CreateDeltaDataset(farm, plot, deltaDays, deltaDaysThreshold, cleanData)
	if err != nil {
		return nil, err
	}
	fmt.Printf("CreateDeltaDataset took %v\n", time.Since(stepStart))

	latitude, longitude, err := sentinel.GetCentroidLatitudeLongitudeFromGeometry(geometry)
	if err != nil {
		return nil, err
	}

	stepStart = time.Now()
	historicalWeather, err := weather.FetchWeather(latitude, longitude, startDate.AddDate(0, -4, 0), endDate, 10)
	if err != nil {
		return nil, err
	}
	fmt.Printf("FetchWeather took %v\n", time.Since(stepStart))

	stepStart = time.Now()
	plotFinalDataset, err := delta1.GetFinalData(deltaDataset, historicalWeather, startDate, endDate, farm, plot)
	if err != nil {
		return nil, err
	}
	fmt.Printf("GetFinalData took %v\n", time.Since(stepStart))

	fmt.Println("Starting ML analysis...")
	stepStart = time.Now()
	result, err := ml.RunModel(model, plotFinalDataset)
	if err != nil {
		return nil, err
	}
	fmt.Printf("RunModel took %v\n", time.Since(stepStart))

	fmt.Printf("Total evaluatePlot execution time: %v\n", time.Since(start))
	return result, nil
}
