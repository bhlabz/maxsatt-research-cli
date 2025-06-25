package delta

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/airbusgeo/godal"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/sentinel"
	"github.com/schollz/progressbar/v3"
)

type Data struct {
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

func deltaDataset(farm, plot string, deltaMin, deltaMax int, cleanDataset map[[2]int]map[time.Time]PixelData) ([]Data, error) {

	var deltaDataset []Data
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

		ascSortedDates := getSortedKeys(data, true)

		for _, startDate := range ascSortedDates {
			minTargetDate := startDate.AddDate(0, 0, deltaMin)
			maxTargetDate := startDate.AddDate(0, 0, deltaMax)

			for _, endDate := range ascSortedDates {
				if endDate.After(maxTargetDate) {
					notFound++
					break
				}
				if endDate.Before(minTargetDate) {
					continue
				}

				ndreValue := data[endDate].NDRE
				ndreStart := data[startDate].NDRE
				ndmiValue := data[endDate].NDMI
				ndmiStart := data[startDate].NDMI
				psriValue := data[endDate].PSRI
				psriStart := data[startDate].PSRI
				ndviValue := data[endDate].NDVI
				ndviStart := data[startDate].NDVI

				timeDiff := int(endDate.Sub(startDate).Hours() / 24)
				ndreDerivative := (ndreValue - ndreStart) / float64(timeDiff)
				ndmiDerivative := (ndmiValue - ndmiStart) / float64(timeDiff)
				psriDerivative := (psriValue - psriStart) / float64(timeDiff)
				ndviDerivative := (ndviValue - ndviStart) / float64(timeDiff)

				deltaDataset = append(deltaDataset, Data{
					Farm:           farm,
					Plot:           plot,
					DeltaMin:       deltaMin,
					DeltaMax:       deltaMax,
					Delta:          timeDiff,
					StartDate:      startDate,
					EndDate:        endDate,
					PixelData:      data[endDate],
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

type InTreatmentPixel struct {
	PixelData
	mostRecentValidPixelDate *time.Time
}

func (p InTreatmentPixel) ListNeighborsByStatus(images map[[2]int]map[time.Time]InTreatmentPixel, date time.Time, statuses ...sentinel.PixelStatus) []InTreatmentPixel {
	//look for valid neighbors
	directions := [][2]int{
		{p.X - 1, p.Y},     // Up
		{p.X + 1, p.Y},     // Down
		{p.X, p.Y - 1},     // Left
		{p.X, p.Y + 1},     // Right
		{p.X - 1, p.Y - 1}, // Up-Left
		{p.X - 1, p.Y + 1}, // Up-Right
		{p.X + 1, p.Y - 1}, // Down-Left
		{p.X + 1, p.Y + 1}, // Down-Right
	}
	var validNeighbors []InTreatmentPixel
	for _, direction := range directions {
		pixelData, directionExists := images[direction]
		if directionExists {
			_, ok := pixelData[date]
			if !ok {
				continue // Skip if the date does not exist for this pixel
			}

			if slices.Contains(statuses, pixelData[date].Status) {
				validNeighbors = append(validNeighbors, pixelData[date])
			}
		}

	}
	return validNeighbors
}

func (p InTreatmentPixel) FindMostRecentPixelsByStatus(datePixel map[time.Time]InTreatmentPixel, currentDate time.Time, statuses ...sentinel.PixelStatus) (*InTreatmentPixel, *time.Time) {
	descSortedDates := getSortedKeys(datePixel, false)

	for _, date := range descSortedDates {
		if date.After(currentDate) || date.Equal(currentDate) {
			continue
		}

		pixelRegressive, ok := datePixel[date]
		if !ok {
			continue
		}

		if slices.Contains(statuses, pixelRegressive.Status) {
			return &pixelRegressive, &date
		}
	}
	return nil, nil
}
func (p InTreatmentPixel) GetNextValidPixel(datePixel map[time.Time]InTreatmentPixel, curretDate time.Time) *InTreatmentPixel {
	var nextValidPixel *InTreatmentPixel
	ascSortedDates := getSortedKeys(datePixel, true)
	for _, date := range ascSortedDates {
		if date.Before(curretDate) || date.Equal(curretDate) {
			continue
		}
		pixelData, ok := datePixel[date]
		if !ok {
			continue
		}
		if pixelData.Status == sentinel.PixelStatusValid {
			nextValidPixel = &pixelData
			break
		}
	}

	return nextValidPixel
}

func getUniqueDates(result map[[2]int]map[time.Time]PixelData) map[time.Time]struct{} {
	allDates := make(map[time.Time]struct{})
	for _, pixels := range result {
		for date := range pixels {
			allDates[date] = struct{}{}
		}
	}

	return allDates
}

func removeInvalidDates(result map[[2]int]map[time.Time]PixelData) map[[2]int]map[time.Time]PixelData {
	allDates := getUniqueDates(result)

	for date := range allDates {
		allInvalid := true
		for _, pixels := range result {
			if pixel, exists := pixels[date]; exists {
				if pixel.Status != sentinel.PixelStatusInvalid {
					allInvalid = false
					break
				}
			}
		}

		// If all pixels are Invalid for this date, remove the date from all entries
		if allInvalid {
			for _, pixels := range result {
				delete(pixels, date)
			}
		}
	}

	return result
}

func CreateCleanDataset(farm, plot string, images map[time.Time]*godal.Dataset) (map[[2]int]map[time.Time]PixelData, error) {
	result, err := CreatePixelDataset(farm, plot, images)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no data available to create the dataset for farm: %s, plot: %s using %d images", farm, plot, len(images))
	}

	result = removeInvalidDates(result)

	result = estimatePixels(result)

	result, err = cleanDataset(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func CreateDeltaDataset(farm, plot string, images map[time.Time]*godal.Dataset, deltaMin, deltaMax int) ([]Data, error) {
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
