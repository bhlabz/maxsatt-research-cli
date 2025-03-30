package main

import (
	"fmt"
	"math"
	"sync"
)

func detectOutliers(data []float64, windowSize int, threshold float64) []float64 {
	cleanedData := make([]float64, len(data))

	for i := range data {
		start := int(math.Max(0, float64(i-windowSize)))
		end := int(math.Min(float64(len(data)), float64(i+windowSize+1)))
		window := data[start:end]

		mean, std := calculateMeanAndStd(window)

		if math.Abs(data[i]-mean) > threshold*std {
			cleanedData[i] = mean
		} else {
			cleanedData[i] = data[i]
		}
	}

	return cleanedData
}

func calculateMeanAndStd(data []float64) (mean, std float64) {
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	mean = sum / float64(len(data))

	variance := 0.0
	for _, v := range data {
		variance += math.Pow(v-mean, 2)
	}
	std = math.Sqrt(variance / float64(len(data)))

	return mean, std
}

func gamSmoothing(values []float64, lam float64) []float64 {
	// Placeholder for GAM smoothing logic
	// Replace this with the actual implementation later
	return values
}

func cleanDataset(pixelDataset []PixelData) []PixelData {
	groupedData := make(map[[2]int][]PixelData)

	// Group data by (x, y)
	for _, data := range pixelDataset {
		key := [2]int{data.X, data.Y}
		groupedData[key] = append(groupedData[key], data)
	}

	var wg sync.WaitGroup
	mu := sync.Mutex{}
	newArray := []PixelData{}

	for key, data := range groupedData {
		wg.Add(1)
		go func(key [2]int, data []PixelData) {
			defer wg.Done()

			var ndre, ndmi, psri, ndvi []float64
			for _, d := range data {
				ndre = append(ndre, d.NDRE)
				ndmi = append(ndmi, d.NDMI)
				psri = append(psri, d.PSRI)
				ndvi = append(ndvi, d.NDVI)
			}

			ndmi = detectOutliers(ndmi, 5, 0.3)
			psri = detectOutliers(psri, 5, 0.3)
			ndre = detectOutliers(ndre, 5, 0.3)
			ndvi = detectOutliers(ndvi, 5, 0.3)

			ndvi = gamSmoothing(ndvi, 0.0005)
			ndmi = gamSmoothing(ndmi, 0.0005)
			psri = gamSmoothing(psri, 0.0005)
			ndre = gamSmoothing(ndre, 0.0005)

			validData := []PixelData{}
			for i := range data {
				if ndmi[i] == 0 || psri[i] == 0 || ndre[i] == 0 || ndvi[i] == 0 {
					continue
				}
				data[i].NDMI = ndmi[i]
				data[i].PSRI = psri[i]
				data[i].NDRE = ndre[i]
				data[i].NDVI = ndvi[i]
				validData = append(validData, data[i])
			}

			mu.Lock()
			newArray = append(newArray, validData...)
			mu.Unlock()
		}(key, data)
	}

	wg.Wait()

	if len(newArray) > 0 {
		return newArray
	} else {
		fmt.Println("No valid data found")
		return nil
	}
}
