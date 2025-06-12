package delta

import (
	"fmt"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/sentinel"
	"slices"
	"time"
)

func createTreatmentImages(images map[[2]int]map[time.Time]PixelData) map[[2]int]map[time.Time]InTreatmentPixel {
	treatmentImages := make(map[[2]int]map[time.Time]InTreatmentPixel)
	//var mostRecentDate time.Time

	for key, datePixel := range images {
		//sortedDates := getSortedKeys(datePixel, false)
		//if len(sortedDates) > 0 && sortedDates[0].After(mostRecentDate) {
		//	mostRecentDate = sortedDates[0]
		//}
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

func groupPixelsByStatus(images map[[2]int]map[time.Time]InTreatmentPixel, date time.Time) (
	treatable []InTreatmentPixel,
	invalid []InTreatmentPixel,
	valid []InTreatmentPixel,
) {
	for _, datePixel := range images {
		pixel, exists := datePixel[date]
		if !exists {
			continue
		}
		switch pixel.Status {
		case sentinel.PixelStatusTreatable:
			treatable = append(treatable, pixel)
		case sentinel.PixelStatusInvalid:
			invalid = append(invalid, pixel)
		case sentinel.PixelStatusValid:
			valid = append(valid, pixel)
		}
	}
	return
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

func estimatePixelIndexes(pixel InTreatmentPixel, deltas []delta) InTreatmentPixel {
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

	return pixel
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
	ascDates = ascDates[1:]

	for key, datePixel := range treatmentImages {
		for _, date := range ascDates {
			pixel, ok := datePixel[date]
			if !ok {
				continue
			}

			if pixel.Status == sentinel.PixelStatusValid {
				continue
			}
			if pixel.Status == sentinel.PixelStatusInvalid {
				mostRecentValidPixel, mostRecentValidPixelDate := pixel.FindMostRecentValidPixel(treatmentImages[key])
				if mostRecentValidPixel == nil {
					continue
				}
				mostRecentValidPixelValidOrTreatableNeighbors := mostRecentValidPixel.ListNeighborsByStatus(treatmentImages, *mostRecentValidPixelDate, sentinel.PixelStatusValid, sentinel.PixelStatusTreatable)
				if len(mostRecentValidPixelValidOrTreatableNeighbors) == 0 {
					continue
				}
				// at least one valid or treatable neighbor must match a current valid or treatable pixel neighbor
				currentValidNeighbors := pixel.ListNeighborsByStatus(treatmentImages, date, sentinel.PixelStatusValid)
				for _, currentNeighbor := range currentValidNeighbors {
					found := false
					for _, mostRecentNeighbor := range mostRecentValidPixelValidOrTreatableNeighbors {
						if mostRecentNeighbor.X == currentNeighbor.X && mostRecentNeighbor.Y == currentNeighbor.Y {
							mostRecentValidPixel.Status = sentinel.PixelStatusTreatable
							mostRecentValidPixel.mostRecentValidPixelDate = mostRecentValidPixelDate
							treatmentImages[key][date] = *mostRecentValidPixel
							found = true
							break
						}
					}
					if found {
						break
					}
				}

			}
		}
	}

	for _, date := range ascDates {
		round := 0
		for {
			treatablePixels, invalidPixels, validPixels := groupPixelsByStatus(treatmentImages, date)
			fmt.Printf("%d - Treatable: %d, Invalid: %d, Valid: %d\n", round, len(treatablePixels), len(invalidPixels), len(validPixels))
			if len(treatablePixels) == 0 {
				break
			}

			for _, pixel := range treatablePixels {
				if pixel.Status != sentinel.PixelStatusTreatable {
					continue
				}

				round++

				deltas := getPixelDeltas(treatmentImages, pixel, date)
				if deltas == nil {
					continue
				}

				treatmentImages[[2]int{pixel.X, pixel.Y}][date] = estimatePixelIndexes(pixel, deltas)
			}
		}
	}

	treatedImages := parseInTreatmentToPixelData(treatmentImages)

	return treatedImages
}
