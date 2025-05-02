package final

import (
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delta"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/weather"
)

// isBetweenDates checks if a date is between startDate and endDate.
func isBetweenDates(date, startDate, endDate time.Time) bool {
	return !date.Before(startDate) && !date.After(endDate)
}

// GetFinalData processes and retrieves climate group data.
func GetFinalData(deltaDataset []delta.DeltaData, historicalWeather weather.HistoricalWeather, startDate, endDate time.Time, farm, plot string, fileName string) ([]FinalData, error) {
	// Construct the file name
	name := farm + "_" + plot + "_" + startDate.Format("2006-01-02")
	if fileName == "" {
		fileName = name + ".csv"
	}

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
		// Truncate to year, month, and day
		truncatedDate := time.Date(endDateRecord.Year(), endDateRecord.Month(), endDateRecord.Day(), 0, 0, 0, 0, endDateRecord.Location())
		dates = append(dates, truncatedDate)
	}

	// Call external functions (placeholders for now)
	climateDataset := weather.CalculateHistoricalWeatherMetricsByDates(dates, historicalWeather)
	return createFinalDataset(filteredDataset, climateDataset, fileName)
}
