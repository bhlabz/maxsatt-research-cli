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

func (p InTreatmentPixel) FindMostRecentValidPixel(datePixel map[time.Time]InTreatmentPixel) (*InTreatmentPixel, *time.Time) {
	ascSortedDates := getSortedKeys(datePixel, true)

	for _, date := range ascSortedDates {
		pixelRegressive := datePixel[date]
		//if pixelRegressive.Status == sentinel.PixelStatusValid {
		//	//mostRecentValidPixelValidNeighbors := pixelRegressive.ListNeighborsByStatus(images, sentinel.PixelStatusValid)
		//	//if len(mostRecentValidPixelValidNeighbors) == 0 {
		//	//	depthRegressive--
		//	//	continue
		//	//}
		//	for _, currentPixelValidNeighbor := range p.ListNeighborsByStatus(images, sentinel.PixelStatusValid) {
		//		for _, mostRecentValidPixelValidNeighbor := range mostRecentValidPixelValidNeighbors {
		//			if currentPixelValidNeighbor.X == mostRecentValidPixelValidNeighbor.X && currentPixelValidNeighbor.Y == mostRecentValidPixelValidNeighbor.Y {
		//				mostRecentValidPixel = &pixelRegressive
		//				break
		//			}
		//		}
		//		if mostRecentValidPixel != nil {
		//			break
		//		}
		//	}
		//	if mostRecentValidPixel != nil {
		//		break
		//	}
		//}
		if pixelRegressive.Status == sentinel.PixelStatusValid {
			return &pixelRegressive, &date
		}
	}
	return nil, nil
}

//func treatPixelData(images map[[2]int][]PixelData) map[[2]int][]PixelData {
//	treatmentImages := make(map[[2]int][]InTreatmentPixel)
//	maxDepth := 1
//
//	for _, sortedPixels := range images {
//		if maxDepth < len(sortedPixels) {
//			maxDepth = len(sortedPixels)
//		}
//		for _, pixel := range sortedPixels {
//			treatmentImages[[2]int{pixel.X, pixel.Y}] = append(treatmentImages[[2]int{pixel.X, pixel.Y}], InTreatmentPixel{
//				PixelData: pixel,
//			})
//		}
//	}
//
//	for depth := 0; depth < maxDepth; depth++ {
//		fmt.Printf("Treating pixels at depth %d\n", depth)
//		treatablePixelsCount := 0
//		untreatablePixelsCount := 0
//		validPixelsCount := 0
//		treatedPixelsCount := 0
//		var date *time.Time
//		for k, sortedPixels := range treatmentImages {
//			if depth >= len(sortedPixels) {
//				continue
//			}
//			if sortedPixels[depth].Status == sentinel.PixelStatusTreatable {
//				if depth == 0 {
//					treatmentImages[k][depth].Status = sentinel.PixelStatusInvalid
//				}
//				treatablePixelsCount++
//			}
//			if sortedPixels[depth].Status == sentinel.PixelStatusValid {
//				validPixelsCount++
//			}
//			if date == nil {
//				date = &sortedPixels[depth].Date
//			}
//		}
//
//		if depth == 0 {
//			continue
//		}
//		anyPixelWasTreated := true
//		rounds := 0
//		fmt.Println(*date)
//		for anyPixelWasTreated {
//			rounds++
//			anyPixelWasTreated = false
//			var readyToTreat []InTreatmentPixel
//			for k, sortedPixels := range treatmentImages {
//				pixel := sortedPixels[depth]
//				//22, 75 on date 2025-03-22 00:00:00 +0000 UTC
//				//if pixel.X == 22 && pixel.Y == 75 && pixel.Status == sentinel.PixelStatusTreatable && *date == time.Date(2025, 3, 22, 0, 0, 0, 0, time.UTC) {
//				//	fmt.Println("here")
//				//}
//				// if pixel is not treatable, skip it
//				if depth >= len(sortedPixels) || pixel.Status == sentinel.PixelStatusValid {
//					continue
//				}
//
//				currentPixelValidNeighbors := pixel.ListNeighborsByStatus(treatmentImages, sentinel.PixelStatusValid)
//				// if the pixel has no valid neighbors, it cannot be treated now. But maybe in future iterations when its neighbors are treated // here
//				if len(currentPixelValidNeighbors) == 0 {
//					continue
//				}
//
//				// look for the most recent valid pixel in the past
//				depthRegressive := depth - 1
//				for depthRegressive >= 0 {
//					pixelRegressive := sortedPixels[depthRegressive]
//					if pixelRegressive.Status == sentinel.PixelStatusValid {
//						// for the pixel to ba valid it must have at least one valid neighbor matching the historicalValidNeighborsDirections from the current pixel
//						mostRecentValidPixelValidNeighbors := pixelRegressive.ListNeighborsByStatus(treatmentImages, sentinel.PixelStatusValid)
//						if len(mostRecentValidPixelValidNeighbors) == 0 {
//							depthRegressive--
//							continue
//						}
//						for _, currentPixelValidNeighbor := range currentPixelValidNeighbors {
//							for _, mostRecentValidPixelValidNeighbor := range mostRecentValidPixelValidNeighbors {
//								if currentPixelValidNeighbor.X == mostRecentValidPixelValidNeighbor.X && currentPixelValidNeighbor.Y == mostRecentValidPixelValidNeighbor.Y {
//									pixel.mostRecentValidPixel = &pixelRegressive
//									treatmentImages[k][depth].mostRecentValidPixel = &pixelRegressive
//									treatmentImages[k][depth].Status = sentinel.PixelStatusTreatable
//									break
//								}
//								if pixel.mostRecentValidPixel != nil {
//									break
//								}
//							}
//							if pixel.mostRecentValidPixel != nil {
//								break
//							}
//						}
//						if pixel.mostRecentValidPixel != nil {
//							break
//						}
//					}
//					depthRegressive--
//				}
//
//				// if no valid pixel found, mark the pixel as invalid and continue. It cannot be treated since we don't have a valid reference value
//				if pixel.mostRecentValidPixel == nil {
//					sortedPixels[depth].Status = sentinel.PixelStatusInvalid
//					continue
//				}
//
//				//readyToTreat = append(readyToTreat, treatmentImages[k][depth])
//			}
//
//			type delta struct {
//				NDRE float64
//				NDMI float64
//				PSRI float64
//				NDVI float64
//			}
//			for _, pixel := range readyToTreat {
//				var deltaArray []delta
//				// get its current valid neighbor and get the equivalent valid neighbor from the mostRecentValidPixel
//				// calculate the delta between the current pixel and the mostRecentValidPixel valid neighbors
//				for _, validNeighbor := range pixel.ListNeighborsByStatus(treatmentImages, sentinel.PixelStatusValid) {
//					if pixel.mostRecentValidPixel == nil {
//						// if no most recent valid pixel found, mark the pixel as invalid
//						treatmentImages[[2]int{pixel.X, pixel.Y}][pixel.Depth].Status = sentinel.PixelStatusInvalid
//						untreatablePixelsCount++
//						continue
//					}
//					for _, mostRecentValidNeighbor := range pixel.mostRecentValidPixel.ListNeighborsByStatus(treatmentImages, sentinel.PixelStatusValid) {
//						if validNeighbor.X == mostRecentValidNeighbor.X && validNeighbor.Y == mostRecentValidNeighbor.Y {
//							deltaArray = append(deltaArray, delta{
//								NDRE: validNeighbor.NDRE - mostRecentValidNeighbor.NDRE,
//								NDMI: validNeighbor.NDMI - mostRecentValidNeighbor.NDMI,
//								PSRI: validNeighbor.PSRI - mostRecentValidNeighbor.PSRI,
//								NDVI: validNeighbor.NDVI - mostRecentValidNeighbor.NDVI,
//							})
//						}
//					}
//				}
//				if len(deltaArray) == 0 {
//					// if no valid neighbor found, mark the pixel as invalid
//					//treatmentImages[[2]int{pixel.X, pixel.Y}][pixel.Depth].Status = sentinel.PixelStatusInvalid
//					untreatablePixelsCount++
//					continue
//				}
//
//				// calculate the average delta
//				averageDelta := delta{
//					NDRE: 0,
//					NDMI: 0,
//					PSRI: 0,
//					NDVI: 0,
//				}
//				for _, d := range deltaArray {
//					averageDelta.NDRE += d.NDRE
//					averageDelta.NDMI += d.NDMI
//					averageDelta.PSRI += d.PSRI
//					averageDelta.NDVI += d.NDVI
//				}
//				averageDelta.NDRE /= float64(len(deltaArray))
//				averageDelta.NDMI /= float64(len(deltaArray))
//				averageDelta.PSRI /= float64(len(deltaArray))
//				averageDelta.NDVI /= float64(len(deltaArray))
//
//				// update the pixel values with the average delta
//				treatedPixel := PixelData{
//					Date:   pixel.Date,
//					X:      pixel.X,
//					Y:      pixel.Y,
//					NDRE:   pixel.mostRecentValidPixel.NDRE + averageDelta.NDRE,
//					NDMI:   pixel.mostRecentValidPixel.NDMI + averageDelta.NDMI,
//					PSRI:   pixel.mostRecentValidPixel.PSRI + averageDelta.PSRI,
//					NDVI:   pixel.mostRecentValidPixel.NDVI + averageDelta.NDVI,
//					Status: sentinel.PixelStatusValid,
//					Color: &color.RGBA{
//						R: 0,
//						G: 0,
//						B: 255,
//						A: 255,
//					},
//				}
//				treatmentImages[[2]int{pixel.X, pixel.Y}][depth] = InTreatmentPixel{
//					PixelData:            treatedPixel,
//					mostRecentValidPixel: pixel.mostRecentValidPixel,
//				}
//				treatedPixelsCount++
//				anyPixelWasTreated = true
//			}
//
//			treatablePixelsCount = 0
//			for _, sortedPixels := range treatmentImages {
//				if depth >= len(sortedPixels) {
//					continue
//				}
//				if sortedPixels[depth].Status == sentinel.PixelStatusTreatable {
//					treatablePixelsCount++
//				}
//
//			}
//
//			fmt.Println("treatedPixelsCount:", treatedPixelsCount, "untreatablePixelsCount:", untreatablePixelsCount, "validPixelsCount:", validPixelsCount, "treatablePixelsCount:", treatablePixelsCount, "rounds:", rounds)
//		}
//
//		for k, sortedPixels := range treatmentImages {
//			if depth >= len(sortedPixels) {
//				continue
//			}
//			if sortedPixels[depth].Status == sentinel.PixelStatusTreatable {
//				sortedPixels[depth].Status = sentinel.PixelStatusInvalid
//				sortedPixels[depth].Color = &color.RGBA{
//					R: 255,
//					G: 0,
//					B: 0,
//					A: 255,
//				}
//				treatmentImages[k][depth] = sortedPixels[depth]
//				untreatablePixelsCount++
//			}
//
//			break
//		}
//	}
//	treatedImages := make(map[[2]int][]PixelData)
//	for k, sortedPixels := range treatmentImages {
//		for _, pixel := range sortedPixels {
//			//if pixel.Status == sentinel.PixelStatusTreatable {
//			//	nbrs := pixel.ListNeighborsByStatus(treatmentImages, sentinel.PixelStatusValid)
//			//	if len(nbrs) != 0 {
//			//		panic(fmt.Sprintf("Treatable pixel found with valid neighbors at %d, %d on date %s", pixel.X, pixel.Y, pixel.Date))
//			//	}
//			//	continue
//			//}
//			if pixel.Status != sentinel.PixelStatusInvalid {
//				if pixel.Status == sentinel.PixelStatusTreatable {
//					pixel.Color = &color.RGBA{
//						R: 255,
//						G: 0,
//						B: 0,
//						A: 255, // Mark treatable pixels with red color
//					}
//				} else {
//					pixel.Color = &color.RGBA{
//						R: 0,
//						G: 255,
//						B: 0,
//						A: 255, // Mark valid pixels with green color
//					}
//				}
//
//				treatedImages[k] = append(treatedImages[k], pixel.PixelData)
//			}
//		}
//	}
//
//	return treatedImages
//}

func CreateCleanDataset(farm, plot string, images map[time.Time]*godal.Dataset) (map[[2]int]map[time.Time]PixelData, error) {
	result, err := CreatePixelDataset(farm, plot, images)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no data available to create the dataset for farm: %s, plot: %s using %d images", farm, plot, len(images))
	}

	//newResult := make(map[[2]int]map[time.Time]PixelData)
	//for _, sortedPixels := range result {
	//	for date, pixel := range sortedPixels {
	//		if pixel.Status == sentinel.PixelStatusValid {
	//			if _, exists := newResult[[2]int{pixel.X, pixel.Y}]; !exists {
	//				newResult[[2]int{pixel.X, pixel.Y}] = make(map[time.Time]PixelData)
	//			}
	//			newResult[[2]int{pixel.X, pixel.Y}][date] = pixel
	//		}
	//	}
	//}
	//result = newResult

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
