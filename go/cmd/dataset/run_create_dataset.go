package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delta"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/final"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/sentinel"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/weather"
	"github.com/gocarina/gocsv"
	"github.com/joho/godotenv"
)

type ValidationRow struct {
	Date     string `csv:"date"`
	Pest     string `csv:"pest"`
	Severity string `csv:"severity"`
	Farm     string `csv:"farm"`
	Plot     string `csv:"plot"`
}

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

func runCreateDataset() {
	fmt.Println("create dataset")
	errors := []string{}
	daysBeforeEvidenceToAnalyze := 5
	deltaDays := 5
	deltaDaysTrashHold := 20
	daysToFetch := deltaDays + deltaDaysTrashHold + daysBeforeEvidenceToAnalyze

	outputFileName := "../../data/model/166.csv"
	validationDataPath := "../../data/training_input/166.csv"

	file, err := os.OpenFile(validationDataPath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	var rows []*ValidationRow
	if err := gocsv.UnmarshalFile(file, &rows); err != nil {
		fmt.Println("Error unmarshalling CSV:", err)
		return
	}

	target := len(rows)
	fmt.Printf("Creating dataset from file %s with %d samples\n", validationDataPath, target)

	for i := 0; i < target; i++ {
		row := rows[i]
		date, err := time.Parse("02/01/06", row.Date)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Error parsing date: %v", err))
			continue
		}
		pest := row.Pest
		severity := row.Severity
		farm := row.Farm
		plot := strings.Split(row.Plot, "-")[1]

		daysBeforeEvidenceToAnalyze = -getDaysBeforeEvidenceToAnalyse(pest, severity)
		geometry, err := sentinel.GetGeometryFromGeoJSON(farm, plot)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Error getting geometry: %v", err))
			continue
		}

		endDate := date.AddDate(0, 0, -(daysBeforeEvidenceToAnalyze - 5))
		startDate := endDate.AddDate(0, 0, -daysToFetch)

		images, err := sentinel.GetImages(geometry, farm, plot, startDate, endDate, 1)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Error getting images: %v", err))
			continue
		}

		latitude, longitude, err := sentinel.GetCentroidLatitudeLongitudeFromGeometry(geometry)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Error getting centroid latitude and longitude: %v", err))
			continue
		}

		historicalWeather, err := weather.FetchWeather(latitude, longitude, startDate.AddDate(0, -4, 0), endDate, 10)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Error getting weather: %v", err))
			continue
		}

		deltaDataset, err := delta.CreateDeltaDataset(farm, plot, images, deltaDays, deltaDaysTrashHold)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Error creating delta dataset: %v", err))
			continue
		}

		samplesAmount := getSamplesAmountFromSeverity(severity, len(deltaDataset))
		bestSamples := getBestSamplesFromDeltaDataset(deltaDataset, samplesAmount, pest)

		_, err = final.GetFinalData(bestSamples, historicalWeather, startDate, endDate, farm, plot, false, outputFileName)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Error getting climate group data: %v", err))
			continue
		}

	}

	fmt.Println(strings.Join(errors, "/n"))
}

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		err := godotenv.Load(".env")
		if err != nil {
			panic(err)
		}
	}
	runCreateDataset()
}
