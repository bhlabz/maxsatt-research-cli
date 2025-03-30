package final

import (
	"os"
	"path/filepath"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/delta"
	"github.com/forest-guardian/forest-guardian-api-poc/weather"
	"github.com/gocarina/gocsv"
)

// isBetweenDates checks if a date is between startDate and endDate.
func isBetweenDates(date, startDate, endDate time.Time) bool {
	return !date.Before(startDate) && !date.After(endDate)
}

// GetFinalData processes and retrieves climate group data.
func GetFinalData(deltaDataset []delta.DeltaData, historicalWeather weather.HistoricalWeather, date time.Time, farm, plot string, deltaDays, deltaDaysTrashHold int, cache bool, fileName string) ([]FinalData, error) {
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

	filteredDataset := make([]delta.DeltaData, 0, len(deltaDataset))
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
	climateDataset := weather.CalculateHistoricalWeatherMetricsByDates(dates, historicalWeather)
	return createFinalDataset(filteredDataset, climateDataset, fileName)
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
