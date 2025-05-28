package delta

import (
	"errors"
	"fmt"
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
	/*
	   - For each treatable pixel
	   	- Search for most recent image where its a valid pixel
	*/

	depth := 1
	for inTreatmentPixelIndex := range depth {
		pq := NewPixelDataPriorityQueue()
		for k, sortedPixels := range pixelDataset {
			if depth != len(sortedPixels) {
				depth = len(sortedPixels)
			}

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
				if pixelDataset[direction][inTreatmentPixelIndex].Date.Equal(sortedPixels[inTreatmentPixelIndex].Date) {
					if pixelDataset[direction][inTreatmentPixelIndex].Status == sentinel.PixelStatusValid {
						pixelDataset[k][inTreatmentPixelIndex].historicalValidNeighborsDirections = append(pixelDataset[k][inTreatmentPixelIndex].historicalValidNeighborsDirections, direction)
						break
					}
				}
			}

			// For each sortedPixel date find the valid neighbors past value (in the best case, this will stop at the most recent date (index 0))
			for _, pixel := range sortedPixels {
				if pixel.Status != sentinel.PixelStatusValid {
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
						if pixel.Date.Equal(mostRecentValidDate) {
							if pixel.Status == sentinel.PixelStatusValid {
								pixelDataset[k][inTreatmentPixelIndex].historicalValidNeighborsDirections = append(pixelDataset[k][i].historicalValidNeighborsDirections, direction)
								found = true
								break
							}
						}
					}

					if !found {
						// remove the direction from the historicalValidNeighborsDirections
						pixelDataset[k][inTreatmentPixelIndex].historicalValidNeighborsDirections = append(pixelDataset[k][inTreatmentPixelIndex].historicalValidNeighborsDirections[:i], pixelDataset[k][inTreatmentPixelIndex].historicalValidNeighborsDirections[i+1:]...)
					}
				}
				// if the pixel has no valid neighbors, continue. Whe cannot treat it
				if len(pixelDataset[k][inTreatmentPixelIndex].historicalValidNeighborsDirections) == 0 {
					continue
				}

				pq.Push(pixelDataset[k][inTreatmentPixelIndex])
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
				continue
			}
			for _, direction := range pixel.historicalValidNeighborsDirections {
				currentNeighborPixel := pixelDataset[direction]
				//todo: calculate the delta and append to a global array so the mean delta can be calculated and added to the pixel being processed
				deltaArray = append(deltaArray, delta{
					NDRE: currentNeighborPixel[0].NDRE - currentNeighborPixel[1].NDRE,
					NDMI: currentNeighborPixel[0].NDRE - currentNeighborPixel[1].NDRE,
					PSRI: currentNeighborPixel[0].NDRE - currentNeighborPixel[1].NDRE,
					NDVI: currentNeighborPixel[0].NDRE - currentNeighborPixel[1].NDRE,
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
			pixelDataset[[2]int{pixel.X, pixel.Y}][0] = pixel

			//todo: reindex pq
			heap.Init(pq)

		}

		for k, pixels := range pixelDataset {
			validPixels := []PixelData{}
			for _, pixel := range pixels {
				if pixel.Status == sentinel.PixelStatusValid {
					validPixels = append(validPixels, pixel)
				}
			}
			if len(validPixels) == 0 {
				delete(pixelDataset, k)
			} else {
				pixelDataset[k] = validPixels
			}
		}
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
