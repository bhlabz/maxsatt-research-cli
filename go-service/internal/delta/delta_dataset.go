package delta

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/airbusgeo/godal"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
	"github.com/gocarina/gocsv"
	"github.com/schollz/progressbar/v3"
)

type DeltaData struct {
	Farm           string    `csv:"farm"`
	Plot           string    `csv:"plot"`
	DeltaMin       int       `csv:"delta_min"`
	DeltaMax       int       `csv:"delta_max"`
	Delta          int       `csv:"delta"`
	StartDate      time.Time `csv:"start_date"`
	EndDate        time.Time `csv:"end_date"`
	X              int       `csv:"x"`
	Y              int       `csv:"y"`
	NDRE           float64   `csv:"ndre"`
	NDMI           float64   `csv:"ndmi"`
	PSRI           float64   `csv:"psri"`
	NDVI           float64   `csv:"ndvi"`
	NDREDerivative float64   `csv:"ndre_derivative"`
	NDMIDerivative float64   `csv:"ndmi_derivative"`
	PSRIDerivative float64   `csv:"psri_derivative"`
	NDVIDerivative float64   `csv:"ndvi_derivative"`
	Label          *string   `csv:"label"`
}

func deltaDataset(farm, plot string, deltaMin, deltaMax int, clearDataset []PixelData) ([]DeltaData, error) {
	// Sort the dataset by date
	sort.Slice(clearDataset, func(i, j int) bool {
		dateI := clearDataset[i].Date
		dateJ := clearDataset[j].Date
		return dateI.Before(dateJ)
	})

	groupedPixels := make(map[string][]PixelData)
	for _, row := range clearDataset {
		key := fmt.Sprintf("%d,%d", row.X, row.Y)
		groupedPixels[key] = append(groupedPixels[key], row)
	}

	deltaDataset := []DeltaData{}
	found := 0
	notFound := 0
	target := len(groupedPixels)

	progressBar := progressbar.Default(int64(target), "Creating delta dataset")

	for _, data := range groupedPixels {
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
					Farm:           farm,
					Plot:           plot,
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
		progressBar.Add(1)
	}

	fmt.Println()

	if len(deltaDataset) == 0 {
		return nil, errors.New("no valid delta data found. The delta dataset is empty")
	}

	return deltaDataset, nil
}

func createDeltaDataset(farm, plot string, images map[time.Time]*godal.Dataset, deltaDays, deltaDaysThreshold int) ([]DeltaData, error) {

	pixelDataset, err := createPixelDataset(farm, plot, images)
	if err != nil {
		return nil, err
	}
	clearDataset, err := cleanDataset(pixelDataset)
	if err != nil {
		return nil, err
	}

	deltaDataset, err := deltaDataset(farm, plot, deltaDays, deltaDays+deltaDaysThreshold, clearDataset)
	if err != nil {
		return nil, err
	}

	return deltaDataset, nil
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func CreateDeltaDataset(farm, plot string, images map[time.Time]*godal.Dataset, deltaDays, deltaDaysThreshold int) ([]DeltaData, error) {
	deltaDatasetFilePath := fmt.Sprintf("%s/data/delta/%s_%s_%d.csv", properties.RootPath(), farm, plot, deltaDays+deltaDays+deltaDaysThreshold)
	if fileExists(deltaDatasetFilePath) {
		var existingDeltaDataset []DeltaData
		file, err := os.Open(deltaDatasetFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open existing delta dataset file: %w", err)
		}
		defer file.Close()

		err = gocsv.UnmarshalFile(file, &existingDeltaDataset)
		if err != nil {
			return nil, fmt.Errorf("failed to read existing delta dataset: %w", err)
		}

		fmt.Printf("Delta dataset already exists at %s.\n", deltaDatasetFilePath)
		return existingDeltaDataset, nil
	}

	deltaData, err := createDeltaDataset(farm, plot, images, deltaDays, deltaDaysThreshold)
	if err != nil {
		return nil, err
	}

	file, err := os.Create(deltaDatasetFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create delta dataset file: %w", err)
	}
	defer file.Close()

	err = gocsv.MarshalFile(deltaData, file)
	if err != nil {
		return nil, fmt.Errorf("failed to write delta dataset to file: %w", err)
	}

	fmt.Printf("Delta dataset saved at %s.\n", deltaDatasetFilePath)
	return deltaData, nil
}
