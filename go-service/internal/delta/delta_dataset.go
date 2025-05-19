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
	return len(pq[i].validPastNeighbors) > len(pq[j].validPastNeighbors)
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
	pq := NewPixelDataPriorityQueue()
	for k, v := range pixelDataset {
		if len(v) == 0 || v[0].Status != sentinel.PixelStatusTreatable {
			continue
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
			currentDateValidPixelsDirections = [][2]int{}
		)
		// Find directions with valid pixels for the currentDate
		for _, direction := range directions {
			if _, ok := pixelDataset[direction]; !ok {
				continue
			}
			for _, pixel := range pixelDataset[direction] {
				if pixel.Date.Equal(v[0].Date) {
					if pixel.Status == sentinel.PixelStatusValid {
						currentDateValidPixelsDirections = append(currentDateValidPixelsDirections, direction)
						break
					}
				}
			}
		}

		for i := 0; i < len(v); i++ {
			if v[i].Status != sentinel.PixelStatusValid {
				continue
			}
			mostRecentValidDate := v[i].Date

			for _, direction := range currentDateValidPixelsDirections {
				if _, ok := pixelDataset[direction]; !ok {
					continue
				}
				for _, pixel := range pixelDataset[direction] {
					if pixel.Date.Equal(mostRecentValidDate) {
						if pixel.Status == sentinel.PixelStatusValid {
							pixelDataset[k][0].validPastNeighbors = append(pixelDataset[k][i].validPastNeighbors, pixel)
							break
						}
					}
				}

			}
			if len(pixelDataset[k][0].validPastNeighbors) == 0 {
				continue
			}

			pq.Push(pixelDataset[k][0])
		}

	}
	return pixelDataset
}

func CreateCleanDataset(farm, plot string, images map[time.Time]*godal.Dataset) (map[[2]int][]PixelData, error) {
	pixelDataset, err := CreatePixelDataset(farm, plot, images)
	if err != nil {
		return nil, err
	}
	cleanDataset, err := cleanDataset(pixelDataset)
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
