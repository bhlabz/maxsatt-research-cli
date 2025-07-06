package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/dataset"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/ml"
	mlpb "github.com/forest-guardian/forest-guardian-api-poc/internal/ml/protobufs"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/sentinel"
)

// PlotPixels handles the UI for plotting pixel values over time
func PlotPixels() {
	fmt.Println("\033[33m\nWarning:\033[0m")
	fmt.Println("\033[33m- A '.geojson' file with the farm name should be present in data/geojsons folder.\033[0m")
	fmt.Println("\033[33m- The '.geojson' file should contain the desired plot in its features identified by plot_id.\n\033[0m")
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("\033[34mEnter the forest name: \033[0m")
	forest, _ := reader.ReadString('\n')
	forest = strings.TrimSpace(forest)

	fmt.Print("\033[34mEnter the plot id: \033[0m")
	plot, _ := reader.ReadString('\n')
	plot = strings.TrimSpace(plot)

	fmt.Print("\033[34mEnter the end date (YYYY-MM-DD): \033[0m")
	endDateInput, _ := reader.ReadString('\n')
	endDateInput = strings.TrimSpace(endDateInput)
	endDate, err := time.Parse("2006-01-02", endDateInput)
	if err != nil {
		fmt.Printf("\n\033[31mInvalid date format: %s. Please use YYYY-MM-DD.\033[0m\n", endDateInput)
		return
	}

	fmt.Print("\033[34mEnter number of days: \033[0m")
	daysInput, _ := reader.ReadString('\n')
	daysInput = strings.TrimSpace(daysInput)
	days, err := strconv.Atoi(daysInput)
	if err != nil || days <= 0 {
		fmt.Printf("\n\033[31mInvalid number of days: %s. Please enter a positive integer.\033[0m\n", daysInput)
		return
	}
	startDate := endDate.AddDate(0, 0, -days)

	geometry, err := sentinel.GetGeometryFromGeoJSON(forest, plot)
	if err != nil {
		fmt.Printf("\n\033[31mError retrieving geometry from GeoJSON: %s\033[0m\n", err.Error())
		return
	}

	images, err := sentinel.GetImages(geometry, forest, plot, startDate, endDate, 1)
	if err != nil {
		fmt.Printf("\n\033[31mError retrieving images: %s\033[0m\n", err.Error())
		return
	}

	data, err := dataset.CreatePixelDataset(forest, plot, images)
	if err != nil {
		fmt.Printf("\n\033[31mError creating pixel dataset: %s\033[0m\n", err.Error())
		return
	}

	cleanData, err := dataset.CreateCleanDataset(forest, plot, data)
	if err != nil {
		fmt.Printf("\n\033[31mError creating clean dataset: %s\033[0m\n", err.Error())
		return
	}

	ndvi := make(map[string]float64)
	ndre := make(map[string]float64)
	ndmi := make(map[string]float64)

	for _, pixelData := range cleanData {
		for date, pixel := range pixelData {
			ndvi[date.Format("2006-01-02")] = pixel.NDVI
			ndre[date.Format("2006-01-02")] = pixel.NDRE
			ndmi[date.Format("2006-01-02")] = pixel.NDMI
		}
	}

	var pixels []*mlpb.Pixel
	for {
		fmt.Print("\033[34mEnter pixel coordinates (x,y) or 'done' to finish: \033[0m")
		var input string
		fmt.Scanln(&input)
		if strings.ToLower(input) == "done" {
			break
		}
		parts := strings.Split(input, ",")
		if len(parts) != 2 {
			fmt.Println("\033[31mInvalid format. Please use x,y\033[0m")
			continue
		}
		x, err := strconv.Atoi(parts[0])
		if err != nil {
			fmt.Println("\033[31mInvalid x coordinate\033[0m")
			continue
		}
		y, err := strconv.Atoi(parts[1])
		if err != nil {
			fmt.Println("\033[31mInvalid y coordinate\033[0m")
			continue
		}
		pixels = append(pixels, &mlpb.Pixel{X: int32(x), Y: int32(y)})
	}

	ml.PlotPixels(forest, plot, ndvi, ndre, ndmi, pixels)

	fmt.Print("\033[34mEnter the ideal delta days for the image analysis: \033[0m")
	var deltaDays int
	fmt.Scanln(&deltaDays)

	fmt.Print("\033[34mEnter the delta days trash hold for the image analysis: \033[0m")
	var deltaDaysThreshold int
	fmt.Scanln(&deltaDaysThreshold)

	deltaData, err := dataset.CreateDeltaDataset(forest, plot, deltaDays, deltaDaysThreshold, cleanData)
	if err != nil {
		fmt.Printf("\n\033[31mError creating delta dataset: %s\033[0m\n", err.Error())
		return
	}

	// Convert deltaData to a format suitable for plotting
	ndreDerivative := make(map[string]float64)
	ndmiDerivative := make(map[string]float64)
	psriDerivative := make(map[string]float64)
	ndviDerivative := make(map[string]float64)

	for _, pixelData := range deltaData {
		for date, pixel := range pixelData {
			ndreDerivative[date.Format("2006-01-02")] = pixel.NDREDerivative
			ndmiDerivative[date.Format("2006-01-02")] = pixel.NDMIDerivative
			psriDerivative[date.Format("2006-01-02")] = pixel.PSRIDerivative
			ndviDerivative[date.Format("2006-01-02")] = pixel.NDVIDerivative
		}
	}

	ml.PlotDeltaPixels(forest, plot, ndreDerivative, ndmiDerivative, psriDerivative, ndviDerivative, pixels)

	fmt.Printf("\n\033[32mSuccessfully created pixel plots!\033[0m\n")
}
