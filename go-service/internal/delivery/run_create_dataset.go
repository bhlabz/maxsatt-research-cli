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
	Forest   string `csv:"forest"`
	Plot     string `csv:"plot"`
}

func getSamplesAmountFromSeverity(_ string, datasetLength int) int {
	if datasetLength <= 2 {
		return datasetLength
	}
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
		date, err := time.Parse("2006-01-02", row.Date)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Error parsing date: %v", err))
			fmt.Println(err.Error())
			continue
		}
		pest := row.Pest
		severity := row.Severity
		forest := row.Forest
		plot := row.Plot

		finalData, err := dataset.GetSavedFinalData(forest, plot, date, deltaMin, deltaMax)
		if err != nil {
			fmt.Println("Error getting saved final dataset: " + err.Error())
		}

		if finalData == nil {

			geometry, err := sentinel.GetGeometryFromGeoJSON(forest, plot)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error getting geometry: %v", err))
				fmt.Println(err.Error())
				continue
			}

			endDate := date.AddDate(0, 0, -daysBeforeEvidenceToAnalyze)
			startDate := endDate.AddDate(0, 0, -daysToFetch)

			images, err := sentinel.GetImages(geometry, forest, plot, startDate, endDate, 1)
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

			data, err := dataset.CreatePixelDataset(forest, plot, images)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error creating pixel dataset: %v", err))
				fmt.Println(err.Error())
				continue
			}
			if len(data) == 0 {
				err = fmt.Errorf("no data available to create the dataset for forest: %s, plot: %s using %d images", forest, plot, len(images))
				errors = append(errors, fmt.Sprintf("Error creating pixel dataset: %v", err))
				fmt.Println(err.Error())
				continue
			}

			cleanData, err := dataset.CreateCleanDataset(forest, plot, data)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error creating clean dataset: %v", err))
				fmt.Println(err.Error())
				continue
			}

			deltaDataset, err := dataset.CreateDeltaDataset(forest, plot, deltaMin, deltaMax, cleanData)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error creating delta dataset: %v", err))
				fmt.Println(err.Error())
				continue
			}

			samplesAmount := getSamplesAmountFromSeverity(severity, len(deltaDataset))
			bestSamples := getBestSamplesFromDeltaDataset(deltaDataset, samplesAmount, pest)

			fmt.Printf("Best samples for pest %s with severity %s: %d samples. dataset with %d samples\n", pest, severity, len(bestSamples), len(deltaDataset))

			createdFinalData, err := dataset.GetFinalData(bestSamples, historicalWeather, startDate, endDate, forest, plot)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error getting climate group data: %v", err))
				fmt.Println(err.Error())
				continue
			}

			err = dataset.SaveFinalData(createdFinalData, date)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error getting climate group data: %v", err))
				fmt.Println(err.Error())
				continue
			}

			finalData = createdFinalData
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
			fmt.Println(err.Error())
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

		fmt.Printf("Processed row %d/%d: Forest=%s, Plot=%s, Pest=%s, Severity=%s, rows=%d\n", i+1, target, forest, plot, pest, severity, len(finalData))

	}

	fmt.Println(strings.Join(errors, "/n"))
	if len(errors) == target {
		return fmt.Errorf("all rows failed during dataset creation: %v", errors)
	}
	if len(errors) > 0 {
		notification.SendDiscordWarnNotification(fmt.Sprintf("Dataset creation completed with %d errors.\n Errors: %s", len(errors), strings.Join(errors, "/n")))
	}
	filePath := fmt.Sprintf("%s/data/model/%s", properties.RootPath(), outputtDataFileName)
	err = deduplicateCSVFile(filePath)
	if err != nil {
		fmt.Printf("[Deduplication] Error during deduplication: %v\n", err)
	}

	fmt.Println("Dataset created successfully")
	return nil
}

// deduplicateCSVFile removes duplicate rows from a CSV file based on selected columns and overwrites the file.
func deduplicateCSVFile(filePath string) error {
	fmt.Printf("[Deduplication] Starting deduplication for file: %s\n", filePath)
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for deduplication: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	var records [][]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read CSV row: %w", err)
		}
		records = append(records, record)
	}

	// Indices of columns to deduplicate on (based on header order)
	colIdx := map[string]int{}
	for i, h := range headers {
		colIdx[h] = i
	}
	// List of columns to deduplicate on
	dedupCols := []string{
		"avg_temperature", "temp_std_dev", "avg_humidity", "humidity_std_dev", "total_precipitation", "dry_days_consecutive",
		"ndre", "ndmi", "psri", "ndvi", "delta_min", "delta_max", "delta", "ndre_derivative", "ndmi_derivative", "psri_derivative", "ndvi_derivative", "label",
	}

	unique := make(map[string]struct{})
	var deduped [][]string
	for _, row := range records {
		var keyParts []string
		for _, col := range dedupCols {
			idx, ok := colIdx[col]
			if !ok || idx >= len(row) {
				keyParts = append(keyParts, "")
			} else {
				keyParts = append(keyParts, row[idx])
			}
		}
		key := strings.Join(keyParts, "||")
		if _, exists := unique[key]; !exists {
			unique[key] = struct{}{}
			deduped = append(deduped, row)
		}
	}

	if len(deduped) == len(records) {
		fmt.Printf("[Deduplication] No duplicates found. Total rows: %d\n", len(deduped))
		return nil
	}

	fmt.Printf("[Deduplication] Removed %d duplicate rows. Clean rows: %d\n", len(records)-len(deduped), len(deduped))

	// Write back to the same file
	tmpPath := filePath + ".tmp"
	outFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file for deduplication: %w", err)
	}
	defer outFile.Close()

	writer := csv.NewWriter(outFile)
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}
	if err := writer.WriteAll(deduped); err != nil {
		return fmt.Errorf("failed to write deduplicated rows: %w", err)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("error flushing CSV writer: %w", err)
	}

	// Replace original file
	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("failed to replace original file with deduplicated file: %w", err)
	}
	fmt.Printf("[Deduplication] Deduplication complete. File updated: %s\n", filePath)
	return nil
}
