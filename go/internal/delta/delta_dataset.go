package delta

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/airbusgeo/godal"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/weather"
)

type DeltaData struct {
	Name           string
	DeltaMin       int
	DeltaMax       int
	Delta          int
	StartDate      time.Time
	EndDate        time.Time
	X              int
	Y              int
	NDRE           float64
	NDMI           float64
	PSRI           float64
	NDVI           float64
	NDREDerivative float64
	NDMIDerivative float64
	PSRIDerivative float64
	NDVIDerivative float64
	Label          *string
}

func parseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}

func deltaDataset(deltaMin, deltaMax int, clearDataset []PixelData) ([]DeltaData, error) {
	// Sort the dataset by date
	sort.Slice(clearDataset, func(i, j int) bool {
		dateI := clearDataset[i].Date
		dateJ := clearDataset[j].Date
		return dateI.Before(dateJ)
	})

	groupedPixels := make(map[string][]PixelData)
	for _, row := range clearDataset {
		key := fmt.Sprintf("%s,%s", row.X, row.Y)
		groupedPixels[key] = append(groupedPixels[key], row)
	}

	deltaDataset := []DeltaData{}
	found := 0
	notFound := 0
	target := len(groupedPixels)

	fmt.Printf("Creating delta dataset: 0/%d\n", target)
	progress := 0

	for _, data := range groupedPixels {
		if len(data) < 3 {
			notFound++
			progress++
			fmt.Printf("\rCreating delta dataset: %d/%d", progress, target)
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

				x, y := data[i].X, data[i].Y
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
					DeltaMin:       deltaMin,
					DeltaMax:       deltaMax,
					Delta:          timeDiff,
					StartDate:      startDate,
					EndDate:        data[j].Date,
					X:              x,
					Y:              y,
					NDRE:           ndreValue - ndreStart,
					NDMI:           ndmiValue - ndmiStart,
					PSRI:           psriValue - psriStart,
					NDVI:           ndviValue - ndviStart,
					NDREDerivative: ndreDerivative,
					NDMIDerivative: ndmiDerivative,
					PSRIDerivative: psriDerivative,
					NDVIDerivative: ndviDerivative,
				})
				found++

				break
			}
		}
		progress++
		fmt.Printf("\rCreating delta dataset: %d/%d", progress, target)
	}

	fmt.Println()

	if len(deltaDataset) == 0 {
		return nil, errors.New("no valid delta data found. The delta dataset is empty")
	}

	return deltaDataset, nil
}

func CreateDeltaDataset(farm, plot string, date time.Time, images map[time.Time]*godal.Dataset, historicalWeather map[time.Time]weather.Weather, deltaDays, deltaDaysThreshold int) ([]DeltaData, error) {
	pixelDataset, err := createPixelDataset(farm, plot, images, historicalWeather)
	if err != nil {
		return nil, err
	}
	clearDataset := cleanDataset(pixelDataset)
	return deltaDataset(deltaDays, deltaDays+deltaDaysThreshold, clearDataset)
}
