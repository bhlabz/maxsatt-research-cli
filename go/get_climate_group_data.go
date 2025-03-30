package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/gocarina/gocsv"
)

// isBetweenDates checks if a date is between startDate and endDate.
func isBetweenDates(date, startDate, endDate time.Time) bool {
	return !date.Before(startDate) && !date.After(endDate)
}

// getClimateGroupData processes and retrieves climate group data.
func getClimateGroupData(deltaDataset []DeltaData, historicalWeather HistoricalWeather, dateStr, farm, plot string, deltaDays, deltaDaysTrashHold int, cache bool, fileName string) ([]FinalData, error) {
	// Parse the input date
	date, err := parseDate(dateStr)
	if err != nil {
		return nil, err
	}

	// Construct the file name
	name := farm + "_" + plot + "_" + date.Format("2006-01-02")
	if fileName == "" {
		fileName = name + ".csv"
	}

	// Check cache
	if cache {
		filePath := filepath.Join("data/climate_group", fileName)
		if _, err := os.Stat(filePath); err == nil {
			return readCSV(filePath)
		}
	}

	// Calculate date range
	endDate := date
	startDate := endDate.AddDate(0, 0, -(deltaDays + deltaDaysTrashHold))

	filteredDataset := make([]DeltaData, 0, len(deltaDataset))
	for _, record := range deltaDataset {
		if isBetweenDates(record.StartDate, startDate, endDate) || isBetweenDates(record.EndDate, startDate, endDate) {
			filteredDataset = append(filteredDataset, record)
		}
	}

	// Get dates from filtered dataset
	dates := []time.Time{}
	for _, record := range filteredDataset {
		endDateRecord := record.EndDate // Access the EndDate field directly
		dates = append(dates, endDateRecord)
	}

	// Call external functions (placeholders for now)
	climateDataset := filterWeatherData(dates, historicalWeather)
	return createClimateGroupDataset(filteredDataset, climateDataset, fileName)
}

func filterWeatherData(dates []time.Time, historicalWeather HistoricalWeather) (historicalWeatherMetrics HistoricalWeatherMetrics) {
	for _, date := range dates {
		if _, exists := historicalWeather[date]; exists {
			historicalWeatherMetrics[date] = calculateMetrics(30, date, historicalWeather)
		}
	}

	return historicalWeatherMetrics
}

func readCSV(filePath string) ([]FinalData, error) {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var records []FinalData
	if err := gocsv.UnmarshalFile(file, &records); err != nil {
		return nil, err
	}

	return records, nil
}
