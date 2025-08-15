package weather

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/cache"
)

type HourlyData struct {
	Time             []string  `json:"time"`
	RelativeHumidity []float64 `json:"relative_humidity_2m"`
}

type DailyData struct {
	Time          []string  `json:"time"`
	Temperature   []float64 `json:"temperature_2m_mean"`
	Precipitation []float64 `json:"precipitation_sum"`
}

type WeatherResponse struct {
	Hourly HourlyData `json:"hourly"`
	Daily  DailyData  `json:"daily"`
}

type Weather struct {
	Precipitation float64
	Temperature   float64
	Humidity      float64
}

type HistoricalWeather map[time.Time]Weather

func calculateMeanHumidity(hourlyData HourlyData) map[string]float64 {
	dailyHumidity := make(map[string][]float64)
	meanHumidity := make(map[string]float64)

	for i, t := range hourlyData.Time {
		h := hourlyData.RelativeHumidity[i]
		date := t[:10] // Extract the date (YYYY-MM-DD)
		dailyHumidity[date] = append(dailyHumidity[date], h)
	}

	for date, humidities := range dailyHumidity {
		var sum float64
		for _, h := range humidities {
			sum += h
		}
		meanHumidity[date] = sum / float64(len(humidities))
	}

	return meanHumidity
}

func FetchWeather(latitude, longitude float64, startDate, endDate time.Time, retries int) (HistoricalWeather, error) {
	weatherCache := cache.NewFileCache[HistoricalWeather]("weather")
	cacheKey := weatherCache.GenerateKey(latitude, longitude, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	// Try to read from cache
	if cached, ok := weatherCache.Get(cacheKey); ok {
		fmt.Printf("Weather cache HIT for key: %s (lat: %.6f, lon: %.6f, %s to %s)\n", 
			cacheKey, latitude, longitude, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
		return cached, nil
	}
	
	fmt.Printf("Weather cache MISS for key: %s (lat: %.6f, lon: %.6f, %s to %s)\n", 
		cacheKey, latitude, longitude, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	url := "https://archive-api.open-meteo.com/v1/archive"
	params := fmt.Sprintf("?latitude=%f&longitude=%f&start_date=%s&end_date=%s&daily=temperature_2m_mean,precipitation_sum&hourly=relative_humidity_2m",
		latitude, longitude, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	var weatherData WeatherResponse
	var attempt int

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	for attempt < retries {
		resp, err := client.Get(url + params)
		if err != nil {
			fmt.Printf("Failed to retrieve data: %v. Retrying... (%d/%d)\n", err, attempt+1, retries)
			time.Sleep(10 * time.Second)
			attempt++
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fmt.Printf("Response headers: Content-Length=%s, Content-Encoding=%s\n", 
				resp.Header.Get("Content-Length"), resp.Header.Get("Content-Encoding"))
			
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Failed to read response body (partial read may have occurred): %v\n", err)
				fmt.Printf("Partial body read (%d bytes): %s\n", len(bodyBytes), string(bodyBytes))
				return nil, fmt.Errorf("failed to read response body: %v", err)
			}
			
			fmt.Printf("Successfully read %d bytes from response body\n", len(bodyBytes))

			err = json.Unmarshal(bodyBytes, &weatherData)
			if err != nil {
				fmt.Printf("Failed to decode JSON response. Body: %s\n", string(bodyBytes))
				return nil, fmt.Errorf("failed to parse response: %v", err)
			}

			// Parse data
			dataParsed := HistoricalWeather{}
			humidity := calculateMeanHumidity(weatherData.Hourly)

			for i, date := range weatherData.Daily.Time {
				parsedDate, err := time.Parse("2006-01-02", date)
				if err != nil {
					return nil, fmt.Errorf("failed to parse date: %v", err)
				}
				dataParsed[parsedDate] = Weather{
					Temperature:   weatherData.Daily.Temperature[i],
					Precipitation: weatherData.Daily.Precipitation[i],
					Humidity:      humidity[date],
				}
			}

			// Write to cache
			if err := weatherCache.Set(cacheKey, dataParsed); err != nil {
				fmt.Printf("Warning: failed to write cache: %v\n", err)
			} else {
				fmt.Printf("Weather cache WRITTEN for key: %s (lat: %.6f, lon: %.6f, %s to %s)\n", 
					cacheKey, latitude, longitude, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
			}

			return dataParsed, nil
		} else {
			fmt.Printf("Failed to retrieve data: %d. Retrying... (%d/%d)\n", resp.StatusCode, attempt+1, retries)
			time.Sleep(10 * time.Second)
			attempt++
		}
	}

	return nil, fmt.Errorf("failed to retrieve data after %d attempts", retries)
}
