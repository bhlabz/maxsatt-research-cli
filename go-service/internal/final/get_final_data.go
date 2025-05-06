package final

import (
	"fmt"
	"os"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delta"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/weather"
	"github.com/gocarina/gocsv"
)

// isBetweenDates checks if a date is between startDate and endDate.
func isBetweenDates(date, startDate, endDate time.Time) bool {
	return !date.Before(startDate) && !date.After(endDate)
}
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func buildFilePath(farm, plot string, date time.Time, deltaMin, deltaMax int) string {
	return fmt.Sprintf("%s/data/final/%s_%s_%s_%d_%d.csv", properties.RootPath(), farm, plot, date.Format("2006-01-02"), deltaMin, deltaMax)
}

func GetSavedFinalData(farm, plot string, date time.Time, deltaMin, deltaMax int) ([]FinalData, error) {
	filePath := buildFilePath(farm, plot, date, deltaMin, deltaMax)
	if fileExists(filePath) {
		var existingFinalData []FinalData
		file, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open existing final data file: %w", err)
		}
		defer file.Close()

		err = gocsv.UnmarshalFile(file, &existingFinalData)
		if err != nil {
			return nil, fmt.Errorf("failed to read existing final data: %w", err)
		}

		fmt.Printf("Final data already exists at %s.\n", filePath)
		return existingFinalData, nil
	}

	return nil, nil
}

func SaveFinalData(finalData []FinalData, date time.Time) error {
	if len(finalData) == 0 {
		return fmt.Errorf("no final data to save")
	}

	filePath := buildFilePath(finalData[0].Farm, finalData[0].Plot, date, finalData[0].DeltaMin, finalData[0].DeltaMax)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create final data file: %w", err)
	}
	defer file.Close()

	err = gocsv.MarshalFile(&finalData, file)
	if err != nil {
		return fmt.Errorf("failed to save final data to file: %w", err)
	}

	fmt.Printf("Final data successfully saved to %s.\n", filePath)
	return nil
}

func GetFinalData(deltaDataset []delta.DeltaData, historicalWeather weather.HistoricalWeather, startDate, endDate time.Time, farm, plot string) ([]FinalData, error) {
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
	return createFinalDataset(filteredDataset, climateDataset)
}
