package weather

import (
	"math"
	"time"
)

type WeatherMetrics struct {
	AvgTemperature float64 `csv:"avg_temperature"`
	TempStdDev     float64 `csv:"temp_std_dev"`
	// TempAnomaly    float64 `csv:"temp_anomaly"`
	AvgHumidity float64 `csv:"avg_humidity"`
	// HumidityAnomaly      float64 `csv:"humidity_anomaly"`
	HumidityStdDev     float64 `csv:"humidity_std_dev"`
	TotalPrecipitation float64 `csv:"total_precipitation"`
	// PrecipitationAnomaly float64 `csv:"precipitation_anomaly"`
	DryDaysConsecutive int `csv:"dry_days_consecutive"`
}

type HistoricalWeatherMetrics map[time.Time]WeatherMetrics

func calculateWeatherMetrics(periodDays int, targetDate time.Time, historicalData HistoricalWeather) WeatherMetrics {
	filteredHistoricalWeather := make(HistoricalWeather)
	var metrics WeatherMetrics
	var temperatures, humidities, precipitations []float64

	// Calculate the start date for the 30-day period
	startDate1Month := targetDate.AddDate(0, 0, -periodDays)

	// Filter historical data for the 30-day period
	for date, record := range historicalData {
		if date.After(startDate1Month) && date.Before(targetDate) {
			filteredHistoricalWeather[date] = record
		}
	}

	// Filter data for the target period
	for _, record := range filteredHistoricalWeather {
		temperatures = append(temperatures, record.Temperature)
		humidities = append(humidities, record.Humidity)
		precipitations = append(precipitations, record.Precipitation)
	}

	// Calculate averages
	metrics.AvgTemperature = mean(temperatures)
	metrics.TempStdDev = stdDev(temperatures)
	metrics.AvgHumidity = mean(humidities)
	metrics.HumidityStdDev = stdDev(humidities)
	metrics.TotalPrecipitation = sum(precipitations)

	// Calculate dry days
	metrics.DryDaysConsecutive = calculateDryDays(precipitations)

	return metrics
}

func calculateDryDays(precipitations []float64) int {
	maxDryDays := 0
	currentDryDays := 0

	for _, precip := range precipitations {
		if precip == 0 {
			currentDryDays++
			if currentDryDays > maxDryDays {
				maxDryDays = currentDryDays
			}
		} else {
			currentDryDays = 0
		}
	}
	return maxDryDays
}

func mean(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := 0.0
	for _, value := range data {
		sum += value
	}
	return sum / float64(len(data))
}

func stdDev(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	meanValue := mean(data)
	var variance float64
	for _, value := range data {
		variance += math.Pow(value-meanValue, 2)
	}
	return math.Sqrt(variance / float64(len(data)))
}

func sum(data []float64) float64 {
	total := 0.0
	for _, value := range data {
		total += value
	}
	return total
}

// func jsonToWeatherData(jsonData string) HistoricalWeather {
// 	var rawData map[string]map[string]float64
// 	err := json.Unmarshal([]byte(jsonData), &rawData)
// 	if err != nil {
// 		log.Fatalf("Error parsing JSON: %v", err)
// 	}

// 	weatherData := make(HistoricalWeather)
// 	for dateStr, values := range rawData {
// 		date, _ := time.Parse("2006-01-02", dateStr)
// 		weatherData[date] = Weather{
// 			Precipitation: values["precipitation"],
// 			Temperature:   values["temperature"],
// 			Humidity:      values["humidity"],
// 		}
// 	}
// 	return weatherData
// }

// func createClimateDataset(dates []time.Time, historicalData HistoricalWeather, outputFileName string) {
// 	var records []WeatherMetrics

// 	for _, date := range dates {
// 		metrics := calculateMetrics(30, date, historicalData)
// 		records = append(records, metrics)
// 	}

// 	file, err := os.Create(outputFileName)
// 	if err != nil {
// 		log.Fatalf("Error creating file: %v", err)
// 	}
// 	defer file.Close()

// 	err = gocsv.MarshalFile(&records, file)
// 	if err != nil {
// 		log.Fatalf("Error writing to file: %v", err)
// 	}
// }

func CalculateHistoricalWeatherMetricsByDates(dates []time.Time, historicalWeather HistoricalWeather) (historicalWeatherMetrics HistoricalWeatherMetrics) {
	historicalWeatherMetrics = make(HistoricalWeatherMetrics)

	for _, date := range dates {
		if _, exists := historicalWeather[date]; exists {
			historicalWeatherMetrics[date] = calculateWeatherMetrics(30, date, historicalWeather)
		}
	}

	return historicalWeatherMetrics
}
