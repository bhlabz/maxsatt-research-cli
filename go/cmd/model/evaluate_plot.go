package main

import (
	"fmt"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delta"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/final"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/sentinel"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/weather"
	"github.com/joho/godotenv"
)

type Result struct {
	X                       int
	Y                       int
	ProbabilityDistribution map[string]any
}

func runModel(finalPlotDataset []final.FinalData) ([]Result, error) {
	return nil, nil
}
func evaluatePlot(farm, plot string, endDate time.Time) error {
	start := time.Now()

	deltaDays := 5
	deltaDaysTrashHold := 20
	getDaysBeforeEvidenceToAnalyse := deltaDays + deltaDaysTrashHold
	startDate := endDate.AddDate(0, 0, -getDaysBeforeEvidenceToAnalyse)
	outputFileName := fmt.Sprintf("%s_%s_%s.csv", farm, plot, endDate.Format("2006-01-02"))

	stepStart := time.Now()
	geometry, err := sentinel.GetGeometryFromGeoJSON(farm, plot)
	if err != nil {
		return err
	}
	fmt.Printf("GetGeometryFromGeoJSON took %v\n", time.Since(stepStart))

	stepStart = time.Now()
	images, err := sentinel.GetImages(geometry, farm, plot, startDate, endDate, 1)
	if err != nil {
		return err
	}
	fmt.Printf("GetImages took %v\n", time.Since(stepStart))

	stepStart = time.Now()
	latitude, longitude, err := sentinel.GetCentroidLatitudeLongitudeFromGeometry(geometry)
	if err != nil {
		return err
	}
	fmt.Printf("GetCentroidLatitudeLongitudeFromGeometry took %v\n", time.Since(stepStart))

	stepStart = time.Now()
	historicalWeather, err := weather.FetchWeather(latitude, longitude, startDate.AddDate(0, -4, 0), endDate, 10)
	if err != nil {
		return err
	}
	fmt.Printf("FetchWeather took %v\n", time.Since(stepStart))

	stepStart = time.Now()
	deltaDataset, err := delta.CreateDeltaDataset(farm, plot, images, deltaDays, deltaDaysTrashHold)
	if err != nil {
		return err
	}
	fmt.Printf("CreateDeltaDataset took %v\n", time.Since(stepStart))

	stepStart = time.Now()
	plotFinalDataset, err := final.GetFinalData(deltaDataset, historicalWeather, startDate, endDate, farm, plot, false, outputFileName)
	if err != nil {
		return err
	}
	fmt.Printf("GetFinalData took %v\n", time.Since(stepStart))

	stepStart = time.Now()
	_, err = runModel(plotFinalDataset)
	if err != nil {
		return err
	}
	fmt.Printf("runModel took %v\n", time.Since(stepStart))

	fmt.Printf("Total evaluatePlot execution time: %v\n", time.Since(start))
	return nil
}

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		err := godotenv.Load("../env")
		if err != nil {
			panic(err)
		}
	}
	evaluatePlot("Boi Preto XI", "055", time.Now().Add(-time.Hour*24*20))
}
