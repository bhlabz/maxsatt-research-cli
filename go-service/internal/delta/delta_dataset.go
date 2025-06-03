package delta

import (
	"errors"
	"fmt"
	"image/color"
	"time"

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

func getValidNeighbors(images map[[2]int][]PixelData, k [2]int, depth int) []PixelData {
	//look for valid neighbors
	directions := [][2]int{
		{k[0] - 1, k[1]},     // Up
		{k[0] + 1, k[1]},     // Down
		{k[0], k[1] - 1},     // Left
		{k[0], k[1] + 1},     // Right
		{k[0] - 1, k[1] - 1}, // Up-Left
		{k[0] - 1, k[1] + 1}, // Up-Right
		{k[0] + 1, k[1] - 1}, // Down-Left
		{k[0] + 1, k[1] + 1}, // Down-Right
	}
	validNeighbors := []PixelData{}
	for _, direction := range directions {
		if pixelData, ok := images[direction]; ok && depth < len(pixelData) && pixelData[depth].Status == sentinel.PixelStatusValid {
			validNeighbors = append(validNeighbors, pixelData[depth])
		}
	}
	return validNeighbors
}

func treatPixelData(images map[[2]int][]PixelData) map[[2]int][]PixelData {

	maxDepth := 1
	for _, sortedPixels := range images {
		if maxDepth < len(sortedPixels) {
			maxDepth = len(sortedPixels)
		}
	}

	for depth := 1; depth < maxDepth; depth++ {
		fmt.Printf("Treating pixels at depth %d\n", depth)
		treatablePixelsCount := 0
		untreatablePixelsCount := 0
		validPixelsCount := 0
		treatedPixelsCount := 0
		var date *time.Time
		for _, sortedPixels := range images {
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
			readyToTreat := []PixelData{}
			notEligiblePixelsCount := 0
			for k, sortedPixels := range images {
				// if pixel is not treatable, skip it
				if depth >= len(sortedPixels) || sortedPixels[depth].Status != sentinel.PixelStatusTreatable {
					continue
				}

				currentPixelValidNeighbors := getValidNeighbors(images, k, depth)
				// if the pixel has no valid neighbors, it cannot be treated now. But maybe in future iterations when its neighbors are treated
				if len(currentPixelValidNeighbors) == 0 {
					// maybe add this pixel will be eligible in the future
					notEligiblePixelsCount++
					continue
				}

				// look for the most recent valid pixel in the past
				depthRegressive := depth - 1
				for depthRegressive >= 0 {
					if sortedPixels[depthRegressive].Status == sentinel.PixelStatusValid {
						// for the pixel to ba valid it must have at least one valid neighbor matching the historicalValidNeighborsDirections from the current pixel
						mostRecentValidPixelValidNeighbors := getValidNeighbors(images, k, depthRegressive)
						if len(mostRecentValidPixelValidNeighbors) == 0 {
							depthRegressive--
							continue
						}
						for _, currentPixelValidNeighbor := range currentPixelValidNeighbors {
							for _, mostRecentValidPixelValidNeighbor := range mostRecentValidPixelValidNeighbors {
								if currentPixelValidNeighbor.X == mostRecentValidPixelValidNeighbor.X && currentPixelValidNeighbor.Y == mostRecentValidPixelValidNeighbor.Y {
									sortedPixels[depth].mostRecentValidPixel = &sortedPixels[depthRegressive]
									break
								}
								if sortedPixels[depth].mostRecentValidPixel != nil {
									break
								}
							}
							if sortedPixels[depth].mostRecentValidPixel != nil {
								break
							}
						}
						depthRegressive--
					}
					depthRegressive--
				}

				// if no valid pixel found, mark the pixel as invalid and continue. It cannot be treated since we don't have a valid reference value
				if sortedPixels[depth].mostRecentValidPixel == nil {
					sortedPixels[depth].Status = sentinel.PixelStatusInvalid
					continue
				}

				readyToTreat = append(readyToTreat, images[k][depth])
			}

			type delta struct {
				NDRE float64
				NDMI float64
				PSRI float64
				NDVI float64
			}
			for _, pixel := range readyToTreat {
				deltaArray := []delta{}
				// get its current valid neighbor and get the equivalent valid neighbor from the mostRecentValidPixel
				// calculate the delta between the current pixel and the mostRecentValidPixel valid neighbors
				for _, validNeighbor := range getValidNeighbors(images, [2]int{pixel.X, pixel.Y}, depth) {
					for _, mostRecentValidNeighbor := range getValidNeighbors(images, [2]int{pixel.mostRecentValidPixel.X, pixel.mostRecentValidPixel.Y}, depth) {
						if validNeighbor.X == mostRecentValidNeighbor.X && validNeighbor.Y == mostRecentValidNeighbor.Y {
							deltaArray = append(deltaArray, delta{
								NDRE: validNeighbor.NDRE - mostRecentValidNeighbor.NDRE,
								NDMI: validNeighbor.NDMI - mostRecentValidNeighbor.NDMI,
								PSRI: validNeighbor.PSRI - mostRecentValidNeighbor.PSRI,
								NDVI: validNeighbor.NDVI - mostRecentValidNeighbor.NDVI,
							})
						}
					}
				}
				if len(deltaArray) == 0 {
					// if no valid neighbor found, mark the pixel as invalid
					pixel.Status = sentinel.PixelStatusInvalid
					untreatablePixelsCount++
					continue
				}

				// calculate the average delta
				averageDelta := delta{
					NDRE: 0,
					NDMI: 0,
					PSRI: 0,
					NDVI: 0,
				}
				for _, d := range deltaArray {
					averageDelta.NDRE += d.NDRE
					averageDelta.NDMI += d.NDMI
					averageDelta.PSRI += d.PSRI
					averageDelta.NDVI += d.NDVI
				}
				averageDelta.NDRE /= float64(len(deltaArray))
				averageDelta.NDMI /= float64(len(deltaArray))
				averageDelta.PSRI /= float64(len(deltaArray))
				averageDelta.NDVI /= float64(len(deltaArray))

				// update the pixel values with the average delta
				tratedPixel := PixelData{
					Date:                 pixel.Date,
					X:                    pixel.X,
					Y:                    pixel.Y,
					NDRE:                 pixel.mostRecentValidPixel.NDRE + averageDelta.NDRE,
					NDMI:                 pixel.mostRecentValidPixel.NDMI + averageDelta.NDMI,
					PSRI:                 pixel.mostRecentValidPixel.PSRI + averageDelta.PSRI,
					NDVI:                 pixel.mostRecentValidPixel.NDVI + averageDelta.NDVI,
					Status:               sentinel.PixelStatusValid,
					mostRecentValidPixel: pixel.mostRecentValidPixel,
					Color: &color.RGBA{
						R: 255,
						G: 255,
						B: 255,
						A: 255,
					},
				}
				images[[2]int{pixel.X, pixel.Y}][depth] = tratedPixel
				treatedPixelsCount++
			}

			treatablePixelsCount = 0
			for _, sortedPixels := range images {
				if depth >= len(sortedPixels) {
					continue
				}
				if sortedPixels[depth].Status == sentinel.PixelStatusTreatable {
					treatablePixelsCount++
				}
			}

			if treatablePixelsCount == notEligiblePixelsCount {
				// if all treatable pixels are not eligible, break the loop and set them to invalid
				for k, sortedPixels := range images {
					if depth >= len(sortedPixels) {
						continue
					}
					if sortedPixels[depth].Status == sentinel.PixelStatusTreatable {
						sortedPixels[depth].Status = sentinel.PixelStatusInvalid
						sortedPixels[depth].Color = &color.RGBA{
							R: 255,
							G: 0,
							B: 0,
							A: 255,
						}
						images[k][depth] = sortedPixels[depth]
						untreatablePixelsCount++
					}

					break
				}

			}

			fmt.Println("treatedPixelsCount:", treatedPixelsCount, "untreatablePixelsCount:", untreatablePixelsCount, "validPixelsCount:", validPixelsCount, "treatablePixelsCount:", treatablePixelsCount, "notEligiblePixelsCount", notEligiblePixelsCount)
		}
	}

	return images
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
