package delta

import (
	"fmt"
	"slices"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/sentinel"
)

func createTreatmentImages(images map[[2]int]map[time.Time]PixelData) map[[2]int]map[time.Time]InTreatmentPixel {
	treatmentImages := make(map[[2]int]map[time.Time]InTreatmentPixel)
	for key, datePixel := range images {
		for date, pixel := range datePixel {
			if _, exists := treatmentImages[key]; !exists {
				treatmentImages[key] = make(map[time.Time]InTreatmentPixel)
			}
			treatmentImages[key][date] = InTreatmentPixel{
				PixelData: pixel,
			}
		}
	}

	return treatmentImages
}

type GroupedPixels struct {
	Treatable []InTreatmentPixel
	Invalid   []InTreatmentPixel
	Valid     []InTreatmentPixel
	Unknown   []InTreatmentPixel
}

func groupPixelsByStatus(images map[[2]int]map[time.Time]InTreatmentPixel, date time.Time) GroupedPixels {
	groupedPixels := GroupedPixels{}
	for _, datePixel := range images {
		pixel, exists := datePixel[date]
		if !exists {
			continue
		}
		switch pixel.Status {
		case sentinel.PixelStatusTreatable:
			groupedPixels.Treatable = append(groupedPixels.Treatable, pixel)
		case sentinel.PixelStatusInvalid:
			groupedPixels.Invalid = append(groupedPixels.Invalid, pixel)
		case sentinel.PixelStatusValid:
			groupedPixels.Valid = append(groupedPixels.Valid, pixel)
		case sentinel.PixelStatusUnknown:
			groupedPixels.Unknown = append(groupedPixels.Unknown, pixel)
		}
	}
	return groupedPixels
}

func parseInTreatmentToPixelData(treatmentImages map[[2]int]map[time.Time]InTreatmentPixel) map[[2]int]map[time.Time]PixelData {
	treatedImages := make(map[[2]int]map[time.Time]PixelData)
	for k, datePixel := range treatmentImages {
		for date, pixel := range datePixel {
			if pixel.Status != sentinel.PixelStatusInvalid {
				if _, exists := treatedImages[k]; !exists {
					treatedImages[k] = make(map[time.Time]PixelData)
				}
				treatedImages[k][date] = pixel.PixelData
			}
		}
	}
	return treatedImages
}

type delta struct {
	NDRE float64
	NDMI float64
	PSRI float64
	NDVI float64
}

func estimatePixelIndexes(date time.Time, pixel InTreatmentPixel, treatmentImages map[[2]int]map[time.Time]InTreatmentPixel) *InTreatmentPixel {
	nextValidPixel := pixel.GetNextValidPixel(treatmentImages[[2]int{pixel.X, pixel.Y}], date)
	if nextValidPixel != nil {
		deltaMean := delta{
			NDRE: (nextValidPixel.NDRE - pixel.NDRE) / 2,
			NDMI: (nextValidPixel.NDMI - pixel.NDMI) / 2,
			PSRI: (nextValidPixel.PSRI - pixel.PSRI) / 2,
			NDVI: (nextValidPixel.NDVI - pixel.NDVI) / 2,
		}

		pixel.NDRE += deltaMean.NDRE
		pixel.NDMI += deltaMean.NDMI
		pixel.PSRI += deltaMean.PSRI
		pixel.NDVI += deltaMean.NDVI
		pixel.Status = sentinel.PixelStatusValid
		return &pixel
	}

	deltas := getPixelDeltas(treatmentImages, pixel, date)
	if deltas == nil {
		return nil
	}

	avgNDRE := 0.0
	avgNDMI := 0.0
	avgPSRI := 0.0
	avgNDVI := 0.0

	for _, d := range deltas {
		avgNDRE += d.NDRE
		avgNDMI += d.NDMI
		avgPSRI += d.PSRI
		avgNDVI += d.NDVI
	}

	avgNDRE /= float64(len(deltas))
	avgNDMI /= float64(len(deltas))
	avgPSRI /= float64(len(deltas))
	avgNDVI /= float64(len(deltas))

	pixel.Status = sentinel.PixelStatusValid
	pixel.NDRE += avgNDRE
	pixel.NDMI += avgNDMI
	pixel.PSRI += avgPSRI
	pixel.NDVI += avgNDVI

	return &pixel
}

func getPixelDeltas(treatmentImages map[[2]int]map[time.Time]InTreatmentPixel, pixel InTreatmentPixel, date time.Time) []delta {
	/*
		- if its treatable, its value is the value of the most recent valid pixel
		- I need to get the valid pixel neighbors and calculate the delta from its past values it they are valid

	*/

	var deltas []delta
	pixelValidNeighbors := pixel.ListNeighborsByStatus(treatmentImages, date, sentinel.PixelStatusValid)
	if len(pixelValidNeighbors) == 0 {
		return nil
	}
	mostRecentValidNeighbors := pixel.ListNeighborsByStatus(treatmentImages, *pixel.mostRecentValidPixelDate, sentinel.PixelStatusValid)
	if len(mostRecentValidNeighbors) == 0 {
		panic("No valid pixels found for most recent valid pixel")
	}
	for _, validNeighbor := range pixelValidNeighbors {
		for _, mostRecentValidPixel := range mostRecentValidNeighbors {
			if validNeighbor.X == mostRecentValidPixel.X && validNeighbor.Y == mostRecentValidPixel.Y {
				deltas = append(deltas, delta{
					NDRE: validNeighbor.NDRE - mostRecentValidPixel.NDRE,
					NDMI: validNeighbor.NDMI - mostRecentValidPixel.NDMI,
					PSRI: validNeighbor.PSRI - mostRecentValidPixel.PSRI,
					NDVI: validNeighbor.NDVI - mostRecentValidPixel.NDVI,
				})
			}
		}
	}
	if len(deltas) == 0 {
		return nil
	}
	return deltas
}

func estimatePixels(images map[[2]int]map[time.Time]PixelData) map[[2]int]map[time.Time]PixelData {
	treatmentImages := createTreatmentImages(images)
	var includedDates []time.Time
	for _, datePixel := range treatmentImages {
		for date := range datePixel {
			if !slices.Contains(includedDates, date) {
				includedDates = append(includedDates, date)
			}
		}
	}

	ascDates := sortDates(includedDates, true)

	for i, date := range ascDates {
		rounds := 0
		statusUpdated := true
		unknownCount := 1
		for statusUpdated && unknownCount > 0 {
			statusUpdated = false
			rounds++
			groupedPixels := groupPixelsByStatus(treatmentImages, date)
			unknownCount = len(groupedPixels.Unknown)
			fmt.Printf("%d - Treatable: %d, Invalid: %d, Valid: %d, Unknown: %d\n", rounds, len(groupedPixels.Treatable), len(groupedPixels.Invalid), len(groupedPixels.Valid), len(groupedPixels.Unknown))
			for _, pixel := range groupedPixels.Unknown {
				key := [2]int{pixel.X, pixel.Y}

				//if its the first image all unknown pixels are invalid
				if i == 0 {
					pixel.Status = sentinel.PixelStatusInvalid
					treatmentImages[key][date] = pixel
					statusUpdated = true
					continue
				}
				mostRecentValidPixel, mostRecentValidPixelDate := pixel.FindMostRecentPixelsByStatus(treatmentImages[key], date, sentinel.PixelStatusValid, sentinel.PixelStatusTreatable)
				if mostRecentValidPixel == nil {
					continue
				}

				mostRecentValidPixelValidOrTreatableNeighbors := mostRecentValidPixel.ListNeighborsByStatus(treatmentImages, *mostRecentValidPixelDate, sentinel.PixelStatusValid, sentinel.PixelStatusTreatable)
				if len(mostRecentValidPixelValidOrTreatableNeighbors) == 0 {
					continue
				}
				// at least one valid or treatable neighbor must match a current valid or treatable pixel neighbor
				isTreatable := false
				currentValidNeighbors := pixel.ListNeighborsByStatus(treatmentImages, date, sentinel.PixelStatusValid, sentinel.PixelStatusTreatable, sentinel.PixelStatusUnknown)

				//if all are unknown, continue
				unknownCount := 0
				for _, currentNeighbor := range currentValidNeighbors {
					if currentNeighbor.Status == sentinel.PixelStatusUnknown {
						unknownCount++
					}
				}
				if unknownCount == len(currentValidNeighbors) {
					continue
				}

				for _, currentNeighbor := range currentValidNeighbors {
					for _, mostRecentNeighbor := range mostRecentValidPixelValidOrTreatableNeighbors {
						if mostRecentNeighbor.X == currentNeighbor.X && mostRecentNeighbor.Y == currentNeighbor.Y {
							mostRecentValidPixel.Status = sentinel.PixelStatusTreatable
							mostRecentValidPixel.mostRecentValidPixelDate = mostRecentValidPixelDate
							treatmentImages[key][date] = *mostRecentValidPixel
							isTreatable = true
							statusUpdated = true
							break
						}
					}
					if isTreatable {
						break
					}
				}

				if !isTreatable {
					pixel.Status = sentinel.PixelStatusInvalid
					treatmentImages[key][date] = pixel
					statusUpdated = true
				}

			}
		}
	}

	fmt.Println("Starting estimation rounds...")

	for _, date := range ascDates {
		round := 0
		statusUpdate := true
		for statusUpdate {
			statusUpdate = false
			groupedPixels := groupPixelsByStatus(treatmentImages, date)
			fmt.Printf("%d - Treatable: %d, Invalid: %d, Valid: %d, Unknown: %d\n", round, len(groupedPixels.Treatable), len(groupedPixels.Invalid), len(groupedPixels.Valid), len(groupedPixels.Unknown))
			if len(groupedPixels.Treatable) == 0 {
				break
			}

			for _, pixel := range groupedPixels.Treatable {
				if pixel.Status != sentinel.PixelStatusTreatable {
					continue
				}

				estimatedPixel := estimatePixelIndexes(date, pixel, treatmentImages)
				if estimatedPixel == nil {
					continue
				}

				treatmentImages[[2]int{pixel.X, pixel.Y}][date] = *estimatedPixel
				statusUpdate = true
			}
			round++
		}
	}

	for _, datePixel := range treatmentImages {
		for date, pixel := range datePixel {
			if pixel.Status == sentinel.PixelStatusTreatable || pixel.Status == sentinel.PixelStatusUnknown {
				pixel.Status = sentinel.PixelStatusInvalid
				treatmentImages[[2]int{pixel.X, pixel.Y}][date] = pixel
			}
		}
	}

	treatedImages := parseInTreatmentToPixelData(treatmentImages)

	return treatedImages
}
