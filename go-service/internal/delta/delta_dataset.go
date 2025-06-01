package delta

import (
	"errors"
	"fmt"
	"image/color"
	"time"

	"container/heap"

	"github.com/airbusgeo/godal"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/sentinel"
	"github.com/schollz/progressbar/v3"
)

type DeltaData struct {
	PixelData
	Farm           string    `csv:"farm"`
	Plot           string    `csv:"plot"`
	DeltaMin       int       `csv:"delta_min"`
	DeltaMax       int       `csv:"delta_max"`
	Delta          int       `csv:"delta"`
	StartDate      time.Time `csv:"start_date"`
	EndDate        time.Time `csv:"end_date"`
	NDREDerivative float64   `csv:"ndre_derivative"`
	NDMIDerivative float64   `csv:"ndmi_derivative"`
	PSRIDerivative float64   `csv:"psri_derivative"`
	NDVIDerivative float64   `csv:"ndvi_derivative"`
	Label          *string   `csv:"label"`
}

func deltaDataset(farm, plot string, deltaMin, deltaMax int, cleanDataset map[[2]int][]PixelData) ([]DeltaData, error) {

	deltaDataset := []DeltaData{}
	found := 0
	notFound := 0
	target := len(cleanDataset)

	progressBar := progressbar.Default(int64(target), "Creating delta dataset")

	for _, data := range cleanDataset {
		if len(data) < 3 {
			notFound++
			progressBar.Add(1)
			continue
		}

		for i := 0; i < len(data)-1; i++ {
			startDate := data[i].Date
			minTargetDate := startDate.AddDate(0, 0, deltaMin)
			maxTargetDate := startDate.AddDate(0, 0, deltaMax)

			for j := i + 1; j < len(data); j++ {
				date := data[j].Date
				if date.After(maxTargetDate) {
					notFound++
					break
				}
				if date.Before(minTargetDate) {
					continue
				}

				ndreValue := data[j].NDRE
				ndreStart := data[i].NDRE
				ndmiValue := data[j].NDMI
				ndmiStart := data[i].NDMI
				psriValue := data[j].PSRI
				psriStart := data[i].PSRI
				ndviValue := data[j].NDVI
				ndviStart := data[i].NDVI

				timeDiff := int(date.Sub(startDate).Hours() / 24)
				ndreDerivative := (ndreValue - ndreStart) / float64(timeDiff)
				ndmiDerivative := (ndmiValue - ndmiStart) / float64(timeDiff)
				psriDerivative := (psriValue - psriStart) / float64(timeDiff)
				ndviDerivative := (ndviValue - ndviStart) / float64(timeDiff)

				deltaDataset = append(deltaDataset, DeltaData{
					Farm:           farm,
					Plot:           plot,
					DeltaMin:       deltaMin,
					DeltaMax:       deltaMax,
					Delta:          timeDiff,
					StartDate:      startDate,
					EndDate:        data[j].Date,
					PixelData:      data[j],
					NDREDerivative: ndreDerivative,
					NDMIDerivative: ndmiDerivative,
					PSRIDerivative: psriDerivative,
					NDVIDerivative: ndviDerivative,
				})
				found++

				break
			}
		}
		progressBar.Add(1)
	}

	fmt.Println()

	if len(deltaDataset) == 0 {
		return nil, errors.New("no valid delta data found. The delta dataset is empty")
	}

	return deltaDataset, nil
}

type PixelDataPriorityQueue []PixelData

func (pq PixelDataPriorityQueue) Len() int { return len(pq) }

func (pq PixelDataPriorityQueue) Less(i, j int) bool {
	// Sort by the length of validPastNeighbors in descending order
	return len(pq[i].historicalValidNeighborsDirections) > len(pq[j].historicalValidNeighborsDirections)
}

func (pq PixelDataPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PixelDataPriorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(PixelData))
}

func (pq *PixelDataPriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

// NewPixelDataPriorityQueue initializes a new priority queue
func NewPixelDataPriorityQueue() *PixelDataPriorityQueue {
	pq := &PixelDataPriorityQueue{}
	heap.Init(pq)
	return pq
}

func treatPixelData(pixelDataset map[[2]int][]PixelData) map[[2]int][]PixelData {

	maxDepth := 1
	for _, sortedPixels := range pixelDataset {
		if maxDepth < len(sortedPixels) {
			maxDepth = len(sortedPixels)
		}
	}
	
	for depth := range maxDepth {
		fmt.Printf("Treating pixels at depth %d\n", depth)
		treatablePixelsCount := 0
		validPixelsCount := 0
		treatedPixelsCount := 0
		var date *time.Time
		for _, sortedPixels := range pixelDataset {
			if depth >= len(sortedPixels) {
				continue
			}
			if sortedPixels[depth].Status == sentinel.PixelStatusTreatable {
				treatablePixelsCount++
			}
			if sortedPixels[depth].Status == sentinel.PixelStatusValid {
				validPixelsCount++
			}
			if date == nil {
				date = &sortedPixels[depth].Date
			}
		}
		fmt.Println(*date)
		for treatablePixelsCount != 0 {
			pq := NewPixelDataPriorityQueue()
			for k, sortedPixels := range pixelDataset {

				var (
					x, y       = k[0], k[1]
					directions = [8][2]int{{x, y + 1},
						{x, y - 1},
						{x - 1, y},
						{x + 1, y},
						{x + 1, y + 1},
						{x - 1, y + 1},
						{x + 1, y - 1},
						{x - 1, y - 1},
					}
				)
				// Find directions with valid pixels for the currentDate
				for _, direction := range directions {
					if _, ok := pixelDataset[direction]; !ok {
						continue
					}
					if pixelDataset[direction][depth].Date.Equal(sortedPixels[depth].Date) {
						if pixelDataset[direction][depth].Status == sentinel.PixelStatusValid {
							pixelDataset[k][depth].historicalValidNeighborsDirections = append(pixelDataset[k][depth].historicalValidNeighborsDirections, direction)
						}
					}
				}

				// For each sortedPixel date find the valid neighbors past value (in the best case, this will stop at the most recent date (index 0))
				for _, pixel := range sortedPixels {
					if pixel.Status != sentinel.PixelStatusTreatable || len(pixel.historicalValidNeighborsDirections) == 0 {
						continue
					}
					mostRecentValidDate := pixel.Date

					// for each currently valid neighbor direction, find the second most recent pixel value (the most recent is the current date, the date whe found a cloudy pixel and we are treating)
					for i, direction := range pixel.historicalValidNeighborsDirections {
						if _, ok := pixelDataset[direction]; !ok {
							continue
						}
						found := false
						// for each valid pixel in the direction, look for the most recent valid pixel
						for _, pixel := range pixelDataset[direction] {
							if len(pixelDataset[k]) > depth && pixel.Date.Equal(mostRecentValidDate) {
								if pixel.Status == sentinel.PixelStatusValid {
									pixelDataset[k][depth].historicalValidNeighborsDirections = append(pixelDataset[k][depth].historicalValidNeighborsDirections, direction)
									found = true
									break
								}
							}
						}

						if !found {
							// remove the direction from the historicalValidNeighborsDirections
							pixelDataset[k][depth].historicalValidNeighborsDirections = append(pixelDataset[k][depth].historicalValidNeighborsDirections[:i], pixelDataset[k][depth].historicalValidNeighborsDirections[i+1:]...)
						}
					}
					// if the pixel has no valid neighbors, continue. Whe cannot treat it
					if len(pixelDataset[k][depth].historicalValidNeighborsDirections) == 0 {
						pixelDataset[k][depth].Status = sentinel.PixelStatusInvalid
						continue
					}

					pq.Push(pixelDataset[k][depth])
				}

			}
			type delta struct {
				NDRE float64
				NDMI float64
				PSRI float64
				NDVI float64
			}
			// Pop the most recent valid pixel
			for pq.Len() > 0 {
				pixel := heap.Pop(pq).(PixelData)
				deltaArray := []delta{}
				if len(pixel.historicalValidNeighborsDirections) < 2 {
					// If the pixel has no valid neighbors, continue. Whe cannot treat it
					pixelDataset[[2]int{pixel.X, pixel.Y}][depth].Status = sentinel.PixelStatusInvalid
					continue
				}
				for _, direction := range pixel.historicalValidNeighborsDirections {
					currentNeighborPixel := pixelDataset[direction]
					deltaArray = append(deltaArray, delta{
						NDRE: currentNeighborPixel[0].NDRE - currentNeighborPixel[1].NDRE,
						NDMI: currentNeighborPixel[0].NDMI - currentNeighborPixel[1].NDMI,
						PSRI: currentNeighborPixel[0].PSRI - currentNeighborPixel[1].PSRI,
						NDVI: currentNeighborPixel[0].NDVI - currentNeighborPixel[1].NDVI,
					})
				}

				// Calculate the mean delta
				meanDelta := delta{}
				for _, d := range deltaArray {
					meanDelta.NDRE += d.NDRE
					meanDelta.NDMI += d.NDMI
					meanDelta.PSRI += d.PSRI
					meanDelta.NDVI += d.NDVI
				}
				meanDelta.NDRE /= float64(len(deltaArray))
				meanDelta.NDMI /= float64(len(deltaArray))
				meanDelta.PSRI /= float64(len(deltaArray))
				meanDelta.NDVI /= float64(len(deltaArray))
				// Add the mean delta to the pixel
				pixel.NDMI += meanDelta.NDMI
				pixel.NDRE += meanDelta.NDRE
				pixel.PSRI += meanDelta.PSRI
				pixel.NDVI += meanDelta.NDVI
				pixel.Status = sentinel.PixelStatusValid
				pixel.historicalValidNeighborsDirections = [][2]int{}
				pixel.Color = &color.RGBA{
					R: uint8(255),
					G: uint8(192),
					B: uint8(203),
				}
				pixel.Date = pixelDataset[[2]int{pixel.X, pixel.Y}][depth].Date
				pixelDataset[[2]int{pixel.X, pixel.Y}][depth] = pixel
				treatedPixelsCount++
				// fmt.Println("Treating pixel at date:", pixel.Date)
				//todo: reindex pq
				heap.Init(pq)

			}
			treatablePixelsCount = 0
			for _, sortedPixels := range pixelDataset {
				if depth >= len(sortedPixels) {
					continue
				}
				if sortedPixels[depth].Status == sentinel.PixelStatusTreatable {
					treatablePixelsCount++
				}
			}
		}

		// for k, pixels := range pixelDataset {
		// 	validPixels := []PixelData{}
		// 	for _, pixel := range pixels {
		// 		if pixel.Status == sentinel.PixelStatusValid {
		// 			validPixels = append(validPixels, pixel)
		// 		}
		// 	}
		// 	if len(validPixels) == 0 {
		// 		delete(pixelDataset, k)
		// 	} else {
		// 		pixelDataset[k] = validPixels
		// 	}
		// }

		fmt.Println("treatedPixelsCount:", treatedPixelsCount, "validPixelsCount:", validPixelsCount, "treatablePixelsCount:", treatablePixelsCount)

		// for _, sortedPixels := range pixelDataset {
		// 	for _, pixel := range sortedPixels {
		// 		if pixel.Color != nil {
		// 			fmt.Println("Pixel:", pixel.X, pixel.Y, "Date:", pixel.Date, "Status:", pixel.Status, "NDMI", pixel.NDMI, "NDRE:", pixel.NDRE, "PSRI:", pixel.PSRI, "NDVI:", pixel.NDVI, "Color:", pixel.Color)
		// 		}
		// 	}
		// }

	}

	return pixelDataset
}

func CreateCleanDataset(farm, plot string, images map[time.Time]*godal.Dataset) (map[[2]int][]PixelData, error) {
	pixelDataset, err := CreatePixelDataset(farm, plot, images)
	if err != nil {
		return nil, err
	}
	if len(pixelDataset) == 0 {
		return nil, fmt.Errorf("no data available to create the dataset for farm: %s, plot: %s using %d images", farm, plot, len(images))
	}
	treatedPixelDataSet := treatPixelData(pixelDataset)
	cleanDataset, err := cleanDataset(treatedPixelDataSet)
	if err != nil {
		return nil, err
	}
	return cleanDataset, nil
}

func CreateDeltaDataset(farm, plot string, images map[time.Time]*godal.Dataset, deltaMin, deltaMax int) ([]DeltaData, error) {
	cleanDataset, err := CreateCleanDataset(farm, plot, images)
	if err != nil {
		return nil, err
	}

	deltaDataset, err := deltaDataset(farm, plot, deltaMin, deltaMax, cleanDataset)
	if err != nil {
		return nil, err
	}

	return deltaDataset, nil
}
