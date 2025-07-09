package dataset

import (
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/utils"
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

	fmt.Printf("Final data with %d rows successfully saved to %s.\n", len(finalData), filePath)
	return nil
}

func GetFinalData(deltaDataset map[[2]int]map[time.Time]DeltaData, historicalWeather weather.HistoricalWeather, startDate, endDate time.Time, farm, plot string) ([]FinalData, error) {
	dates := make([]time.Time, 0)
	for date := range deltaDataset {
		for date := range deltaDataset[date] {
			if isBetweenDates(date, startDate, endDate) && !slices.Contains(dates, date) {
				dates = append(dates, date)
			}
		}
	}

	lastDate := utils.SortDates(dates, false)[0]

	climateDataset := weather.CalculateHistoricalWeatherMetricsByDates(dates, historicalWeather)

	samples := make(map[[2]int]DeltaData)
	for key, data := range deltaDataset {
		for date, sample := range data {
			if date.Equal(lastDate) {
				samples[key] = sample
			}
		}

	}
	return createFinalDataset(samples, climateDataset)
}
