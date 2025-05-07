package delivery

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delta"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/final"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/notification"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/sentinel"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/weather"
	"github.com/gocarina/gocsv"
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

func CreateDataset(inputDataFileName, outputtDataFileName string, deltaDays, deltaDaysTrashHold int) error {
	fmt.Println("create dataset")
	errors := []string{}
	daysBeforeEvidenceToAnalyze := 5
	daysToFetch := deltaDays + deltaDaysTrashHold + daysBeforeEvidenceToAnalyze
	deltaMin, deltaMax := deltaDays, deltaDays+deltaDaysTrashHold

	validationDataPath := fmt.Sprintf("%s/data/training_input/%s", properties.RootPath(), inputDataFileName)

	file, err := os.OpenFile(validationDataPath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return err
	}
	defer file.Close()

	var rows []*ValidationRow
	if err := gocsv.UnmarshalFile(file, &rows); err != nil {
		fmt.Println("Error unmarshalling CSV:", err)
		return err
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

		finalData, err := final.GetSavedFinalData(farm, plot, date, deltaMin, deltaMax)
		if err != nil {
			fmt.Println("Error getting saved final dataset: " + err.Error())
		}

		if finalData == nil {

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

			deltaDataset, err := delta.CreateDeltaDataset(farm, plot, images, deltaMin, deltaMax)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error creating delta dataset: %v", err))
				continue
			}

			samplesAmount := getSamplesAmountFromSeverity(severity, len(deltaDataset))
			bestSamples := getBestSamplesFromDeltaDataset(deltaDataset, samplesAmount, pest)

			finalData, err := final.GetFinalData(bestSamples, historicalWeather, startDate, endDate, farm, plot)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error getting climate group data: %v", err))
				continue
			}

			err = final.SaveFinalData(finalData, date)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error getting climate group data: %v", err))
				continue
			}
		}
		filePath := fmt.Sprintf("%s/data/model/%s", properties.RootPath(), outputtDataFileName)
		fileExists := false

		// Check if the file already exists
		if _, err := os.Stat(filePath); err == nil {
			fileExists = true
		}

		file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Error opening dataset: %v", err))
			continue
		}
		defer file.Close()

		if fileExists {
			_, err = file.Seek(0, io.SeekEnd)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error seeking to end of file: %v", err))
				continue
			}
		}

		writer := csv.NewWriter(file)
		defer writer.Flush()

		// Write the header only if the file does not already exist
		if !fileExists {
			if err := gocsv.MarshalCSV(&finalData, writer); err != nil {
				errors = append(errors, fmt.Sprintf("Error writing header to CSV file: %v", err))
				continue
			}
			continue
		}
		// Write the data rows
		if err := gocsv.MarshalCSVWithoutHeaders(&finalData, writer); err != nil {
			errors = append(errors, fmt.Sprintf("Error writing to CSV file: %v", err))
			continue
		}

	}

	fmt.Println(strings.Join(errors, "/n"))
	if len(errors) == target {
		return fmt.Errorf("all rows failed during dataset creation: %v", errors)
	}
	if len(errors) > 0 {
		notification.SendDiscordWarnNotification(fmt.Sprintf("Dataset creation completed with %d errors.\n Errors: %s", len(errors), strings.Join(errors, "/n")))
	}
	fmt.Println("Dataset created successfully")
	return nil
}
