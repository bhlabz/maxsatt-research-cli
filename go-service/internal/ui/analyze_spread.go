package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/dataset"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/sentinel"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/spread"
	"github.com/forest-guardian/forest-guardian-api-poc/output"
)

// AnalyzeSpread handles the UI for analyzing forest plot image deforestation spread over time
func AnalyzeSpread() {
	fmt.Println("\033[33m\nWarning:\033[0m")
	fmt.Println("\033[33m- A '.geojson' file with the forest name should be present in data/geojsons folder.\033[0m")
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

	groupedCleanData := make(map[time.Time][]dataset.PixelData)
	for _, sortedPixels := range cleanData {
		for date, pixel := range sortedPixels {
			groupedCleanData[date] = append(groupedCleanData[date], pixel)
		}
	}

	deltaData, err := dataset.CreateDeltaDataset(forest, plot, 1, 20, cleanData)
	if err != nil {
		fmt.Printf("\n\033[31mError creating delta dataset: %s\033[0m\n", err.Error())
		return
	}

	fmt.Print("\033[34mEnter number of days to cluster: \033[0m")
	daysToClusterInput, _ := reader.ReadString('\n')
	daysToClusterInput = strings.TrimSpace(daysToClusterInput)
	daysToCluster, err := strconv.Atoi(daysToClusterInput)
	if err != nil || daysToCluster <= 0 {
		fmt.Printf("\n\033[31mInvalid number of days to cluster: %s. Please enter a positive integer.\033[0m\n", daysToClusterInput)
		return
	}

	spreadResult, err := spread.PestSpread(deltaData, daysToCluster)
	if err != nil {
		fmt.Printf("\n\033[31mError spreading pest: %s\033[0m\n", err.Error())
		return
	}

	for date, pixels := range spreadResult {
		output.CreatePestSpreadImage(pixels, forest, plot, date)
	}

	imagesPath := fmt.Sprintf("%s/data/result/%s/%s/spread", properties.RootPath(), forest, plot)

	err = output.CreateVideoFromDirectory(imagesPath+"/images", imagesPath+"/videos/result")
	if err != nil {
		fmt.Printf("\n\033[31mError creating video: %s\033[0m\n", err.Error())
		return
	}

	fmt.Printf("\n\033[32mSuccessfully created pest spread analysis video!\033[0m\n")
}
