package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delta"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/final"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/sentinel"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/weather"
	"github.com/joho/godotenv"
)

func getDaysBeforeEvidenceToAnalyse(pest, severity string) int {
	return 5
}

func getSamplesAmountFromSeverity(severity string, datasetLength int) int {
	return datasetLength / 2
}

func getBestSamplesFromDeltaDataset(deltaDataset []delta.DeltaData, samplesAmount int, label string) []delta.DeltaData {
	// Sort the deltaDataset based on the specified derivatives
	sort.Slice(deltaDataset, func(i, j int) bool {
		if deltaDataset[i].NDREDerivative != deltaDataset[j].NDREDerivative {
			return deltaDataset[i].NDREDerivative < deltaDataset[j].NDREDerivative
		}
		if deltaDataset[i].NDMIDerivative != deltaDataset[j].NDMIDerivative {
			return deltaDataset[i].NDMIDerivative < deltaDataset[j].NDMIDerivative
		}
		if deltaDataset[i].NDVIDerivative != deltaDataset[j].NDVIDerivative {
			return deltaDataset[i].NDVIDerivative < deltaDataset[j].NDVIDerivative
		}
		return deltaDataset[i].PSRIDerivative > deltaDataset[j].PSRIDerivative
	})

	// Add name and pest (label) to each sample
	for i := range deltaDataset {
		deltaDataset[i].Label = &label
	}

	// Select the top samplesAmount rows
	if samplesAmount > len(deltaDataset) {
		samplesAmount = len(deltaDataset)
	}
	return deltaDataset[:samplesAmount]
}

type Result struct {
	X                       int
	Y                       int
	ProbabilityDistribution map[string]any
}

func runModel(finalPlotDataset []final.FinalData) ([]Result, error) {
	return nil, nil
}

func evaluatePlot(farm, plot string, endDate time.Time) error {
	deltaDays := 5
	deltaDaysTrashHold := 20
	getDaysBeforeEvidenceToAnalyse := deltaDays + deltaDaysTrashHold
	startDate := endDate.AddDate(0, 0, -getDaysBeforeEvidenceToAnalyse)
	outputFileName := fmt.Sprintf("%s_%s_%s.csv", farm, plot, endDate.Format("2006-01-02"))
	geometry, err := sentinel.GetGeometryFromGeoJSON(farm, plot)
	if err != nil {
		return err

	}

	images, err := sentinel.GetImages(geometry, farm, plot, startDate, endDate, 1)
	if err != nil {
		return err

	}

	latitude, longitude, err := sentinel.GetCentroidLatitudeLongitudeFromGeometry(geometry)
	if err != nil {
		return err

	}

	historicalWeather, err := weather.FetchWeather(latitude, longitude, startDate.AddDate(0, -4, 0), endDate, 10)
	if err != nil {
		return err
	}

	deltaDataset, err := delta.CreateDeltaDataset(farm, plot, images, deltaDays, deltaDaysTrashHold)
	if err != nil {
		return err
	}

	plotFinalDataset, err := final.GetFinalData(deltaDataset, historicalWeather, startDate, endDate, farm, plot, false, outputFileName)
	if err != nil {
		return err
	}

	_, err = runModel(plotFinalDataset)
	if err != nil {
		return err
	}

	return nil

}

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		err := godotenv.Load(".env")
		if err != nil {
			panic(err)
		}
	}
}
