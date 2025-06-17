package delivery

import (
	"fmt"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delta"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/final"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/ml"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/sentinel"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/weather"
)

func EvaluatePlotCleanData(farm, plot string, endDate time.Time) ([]delta.PixelData, error) {
	startDate := endDate.AddDate(0, 0, -50)

	geometry, err := sentinel.GetGeometryFromGeoJSON(farm, plot)
	if err != nil {
		return nil, err
	}

	images, err := sentinel.GetImages(geometry, farm, plot, startDate, endDate, 1)
	if err != nil {
		return nil, err
	}

	cleanDataset, err := delta.CreateCleanDataset(farm, plot, images)
	if err != nil {
		return nil, err
	}

	groupedData := make(map[time.Time][]delta.PixelData)
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

func EvaluatePlotDeltaData(deltaDays, deltaDaysThreshold int, farm, plot string, endDate time.Time) ([]delta.Data, error) {

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

	deltaDataset, err := delta.CreateDeltaDataset(farm, plot, images, deltaDays, deltaDaysThreshold)
	if err != nil {
		return nil, err
	}

	return deltaDataset, nil
}

func EvaluatePlotFinalData(model, farm, plot string, endDate time.Time) ([]ml.PixelResult, error) {
	start := time.Now()
	var discard1, discard2, discard3, discard4 int
	var deltaDays, deltaDaysThreshold int

	_, err := fmt.Sscanf(model, "%d_%d-%d-%d_%d_%d.csv",
		&discard1, &discard2, &discard3, &discard4,
		&deltaDays, &deltaDaysThreshold)
	if err != nil {
		return nil, fmt.Errorf("failed to parse model string: %w", err)
	}

	getDaysBeforeEvidenceToAnalyse := deltaDays + deltaDaysThreshold
	startDate := endDate.AddDate(0, 0, -getDaysBeforeEvidenceToAnalyse)

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
	deltaDataset, err := delta.CreateDeltaDataset(farm, plot, images, deltaDays, deltaDaysThreshold)
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
	plotFinalDataset, err := final.GetFinalData(deltaDataset, historicalWeather, startDate, endDate, farm, plot)
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
