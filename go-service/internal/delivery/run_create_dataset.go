package delivery

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/dataset"
	delta1 "github.com/forest-guardian/forest-guardian-api-poc/internal/dataset"
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

func getSamplesAmountFromSeverity(severity string, datasetLength int) int {
	return datasetLength / 2
}

func getBestSamplesFromDeltaDataset(deltaDataset map[[2]int]map[time.Time]dataset.DeltaData, samplesAmount int, label string) map[[2]int]map[time.Time]dataset.DeltaData {
	// Sort the deltaDataset based on the specified derivatives
	deltaDatasetSlice := []dataset.DeltaData{}
	for _, data := range deltaDataset {
		for _, sample := range data {
			deltaDatasetSlice = append(deltaDatasetSlice, sample)
		}
	}

	sort.Slice(deltaDatasetSlice, func(i, j int) bool {
		if deltaDatasetSlice[i].NDREDerivative != deltaDatasetSlice[j].NDREDerivative {
			return deltaDatasetSlice[i].NDREDerivative < deltaDatasetSlice[j].NDREDerivative
		}
		if deltaDatasetSlice[i].NDMIDerivative != deltaDatasetSlice[j].NDMIDerivative {
			return deltaDatasetSlice[i].NDMIDerivative < deltaDatasetSlice[j].NDMIDerivative
		}
		if deltaDatasetSlice[i].NDVIDerivative != deltaDatasetSlice[j].NDVIDerivative {
			return deltaDatasetSlice[i].NDVIDerivative < deltaDatasetSlice[j].NDVIDerivative
		}
		return deltaDatasetSlice[i].PSRIDerivative > deltaDatasetSlice[j].PSRIDerivative
	})

	// Add name and pest (label) to each sample
	for i := range deltaDatasetSlice {
		deltaDatasetSlice[i].Label = &label
	}

	// Select the top samplesAmount rows
	if samplesAmount > len(deltaDatasetSlice) {
		samplesAmount = len(deltaDatasetSlice)
	}
	acceptedCut := deltaDatasetSlice[:samplesAmount]

	newDeltaDataset := make(map[[2]int]map[time.Time]dataset.DeltaData)
	for _, sample := range acceptedCut {
		key := [2]int{sample.X, sample.Y}
		if _, exists := newDeltaDataset[key]; !exists {
			newDeltaDataset[key] = make(map[time.Time]dataset.DeltaData)
		}
		newDeltaDataset[key][sample.StartDate] = sample
	}
	return newDeltaDataset
}

func CreateDataset(inputDataFileName, outputtDataFileName string, deltaDays, deltaDaysTrashHold, daysBeforeEvidenceToAnalyze int) error {
	fmt.Println("create dataset")
	errors := []string{}
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
		var err error
		defer func() {
			fmt.Println("Error:", err)
		}()
		row := rows[i]
		date, err := time.Parse("02/01/06", row.Date)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Error parsing date: %v", err))
			fmt.Println(err.Error())
			continue
		}
		pest := row.Pest
		severity := row.Severity
		farm := row.Farm
		plot := strings.Split(row.Plot, "-")[1]

		finalData, err := delta1.GetSavedFinalData(farm, plot, date, deltaMin, deltaMax)
		if err != nil {
			fmt.Println("Error getting saved final dataset: " + err.Error())
		}

		if finalData == nil {

			geometry, err := sentinel.GetGeometryFromGeoJSON(farm, plot)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error getting geometry: %v", err))
				fmt.Println(err.Error())
				continue
			}

			endDate := date.AddDate(0, 0, -daysBeforeEvidenceToAnalyze)
			startDate := endDate.AddDate(0, 0, -daysToFetch)

			images, err := sentinel.GetImages(geometry, farm, plot, startDate, endDate, 1)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error getting images: %v", err))
				fmt.Println(err.Error())
				continue
			}

			latitude, longitude, err := sentinel.GetCentroidLatitudeLongitudeFromGeometry(geometry)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error getting centroid latitude and longitude: %v", err))
				fmt.Println(err.Error())
				continue
			}

			historicalWeather, err := weather.FetchWeather(latitude, longitude, startDate.AddDate(0, -4, 0), endDate, 10)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error getting weather: %v", err))
				fmt.Println(err.Error())
				continue
			}

			data, err := dataset.CreatePixelDataset(farm, plot, images)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error creating pixel dataset: %v", err))
				fmt.Println(err.Error())
				continue
			}
			if len(data) == 0 {
				err = fmt.Errorf("no data available to create the dataset for farm: %s, plot: %s using %d images", farm, plot, len(images))
				errors = append(errors, fmt.Sprintf("Error creating pixel dataset: %v", err))
				fmt.Println(err.Error())
				continue
			}

			cleanData, err := dataset.CreateCleanDataset(farm, plot, data)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error creating clean dataset: %v", err))
				fmt.Println(err.Error())
				continue
			}

			deltaDataset, err := dataset.CreateDeltaDataset(farm, plot, deltaMin, deltaMax, cleanData)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error creating delta dataset: %v", err))
				fmt.Println(err.Error())
				continue
			}

			samplesAmount := getSamplesAmountFromSeverity(severity, len(deltaDataset))
			bestSamples := getBestSamplesFromDeltaDataset(deltaDataset, samplesAmount, pest)

			finalData, err := delta1.GetFinalData(bestSamples, historicalWeather, startDate, endDate, farm, plot)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error getting climate group data: %v", err))
				fmt.Println(err.Error())
				continue
			}

			err = delta1.SaveFinalData(finalData, date)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error getting climate group data: %v", err))
				fmt.Println(err.Error())
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
				fmt.Println(err.Error())
				continue
			}
		}

		writer := csv.NewWriter(file)
		defer writer.Flush()

		// Write the header only if the file does not already exist
		if !fileExists {
			if err := gocsv.MarshalCSV(&finalData, writer); err != nil {
				errors = append(errors, fmt.Sprintf("Error writing header to CSV file: %v", err))
				fmt.Println(err.Error())
				continue
			}
			continue
		}
		// Write the data rows
		if err := gocsv.MarshalCSVWithoutHeaders(&finalData, writer); err != nil {
			errors = append(errors, fmt.Sprintf("Error writing to CSV file: %v", err))
			fmt.Println(err.Error())
			continue
		}

		fmt.Printf("Processed row %d/%d: Farm=%s, Plot=%s, Pest=%s, Severity=%s\n", i+1, target, farm, plot, pest, severity)

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
