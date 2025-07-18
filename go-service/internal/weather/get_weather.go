package weather

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
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
	// Generate cache key
	cacheKeyRaw := fmt.Sprintf("%f_%f_%s_%s", latitude, longitude, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	h := sha1.New()
	h.Write([]byte(cacheKeyRaw))
	cacheKey := hex.EncodeToString(h.Sum(nil))
	cacheDir := filepath.Join(properties.RootPath()+"/data", "weather")
	cacheFile := filepath.Join(cacheDir, cacheKey+".json")

	// Try to read from cache
	if data, err := ioutil.ReadFile(cacheFile); err == nil {
		var cached HistoricalWeather
		if err := json.Unmarshal(data, &cached); err == nil {
			return cached, nil
		}
		// If unmarshal fails, fall through to fetch
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %v", err)
	}

	url := "https://archive-api.open-meteo.com/v1/archive"
	params := fmt.Sprintf("?latitude=%f&longitude=%f&start_date=%s&end_date=%s&daily=temperature_2m_mean,precipitation_sum&hourly=relative_humidity_2m",
		latitude, longitude, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	var weatherData WeatherResponse
	var attempt int

	for attempt < retries {
		resp, err := http.Get(url + params)
		if err != nil {
			fmt.Printf("Failed to retrieve data: %v. Retrying... (%d/%d)\n", err, attempt+1, retries)
			time.Sleep(10 * time.Second)
			attempt++
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			err = json.NewDecoder(resp.Body).Decode(&weatherData)
			if err != nil {
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
			if data, err := json.Marshal(dataParsed); err == nil {
				ioutil.WriteFile(cacheFile, data, 0644)
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
