package final

import (
	"fmt"
	"sync"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delta"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/weather"
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
	delta.Data
	CreatedAt time.Time `csv:"created_at"`
}

func createFinalDataset(samples []delta.Data, weatherData weather.HistoricalWeatherMetrics) ([]FinalData, error) {
	var mergedData []FinalData
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(samples))

	for _, sample := range samples {
		wg.Add(1)
		go func(sample delta.Data) {
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
				Data:           sample,
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

	return mergedData, nil
}
