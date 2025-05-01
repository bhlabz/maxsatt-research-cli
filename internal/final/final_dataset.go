package final

import (
	"encoding/csv"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delta"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/weather"
	"github.com/gocarina/gocsv"
)

type Sample struct {
	StartDate time.Time
	EndDate   time.Time
	X         int
	Y         int
	Label     string
}

type FinalData struct {
	weather.WeatherMetrics
	delta.DeltaData
	CreatedAt time.Time `csv:"created_at"`
}

func createFinalDataset(samples []delta.DeltaData, weatherData weather.HistoricalWeatherMetrics, outputFileName string) ([]FinalData, error) {
	var mergedData []FinalData
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(samples))

	for _, sample := range samples {
		wg.Add(1)
		go func(sample delta.DeltaData) {
			defer wg.Done()

			weatherRow := weather.WeatherMetrics{}
			found := false

			for date, data := range weatherData {
				startDate := sample.StartDate
				endDate := sample.EndDate
				if !date.Before(startDate) && !date.After(endDate) {
					weatherRow = data
					found = true
					break
				}
			}

			if !found {
				errChan <- fmt.Errorf("weather not found for %s to %s", sample.StartDate, sample.EndDate)
				return
			}

			mergedRow := FinalData{
				WeatherMetrics: weatherRow,
				DeltaData:      sample,
				CreatedAt:      time.Now(),
			}

			mu.Lock()
			mergedData = append(mergedData, mergedRow)
			mu.Unlock()
		}(sample)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return nil, <-errChan
	}

	if outputFileName != "" {
		file, err := os.Create(outputFileName)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		// Use gocsv to write the data
		if err := gocsv.MarshalFile(&mergedData, file); err != nil {
			return nil, fmt.Errorf("failed to write CSV using gocsv: %w", err)
		}
	}

	return mergedData, nil
}
