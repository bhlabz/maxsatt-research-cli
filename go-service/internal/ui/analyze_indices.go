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
	"github.com/forest-guardian/forest-guardian-api-poc/output"
)

// AnalyzeIndices handles the UI for analyzing forest plot image indices over time
func AnalyzeIndices() {
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

	groupedCleanData := make(map[time.Time][]dataset.PixelData)
	for _, sortedPixels := range cleanData {
		for date, pixel := range sortedPixels {
			groupedCleanData[date] = append(groupedCleanData[date], pixel)
		}
	}

	for date, pixels := range groupedCleanData {
		output.CreateCleanDataImage(pixels, forest, plot, date)
	}

	imagesPath := fmt.Sprintf("%s/data/result/%s/%s/clean", properties.RootPath(), forest, plot)

	err = output.CreateVideoFromDirectory(imagesPath+"/images/NDVI", fmt.Sprintf("%s/videos/NDVI-%s-%s", imagesPath, startDate.Format("2006_01_02"), endDate.Format("2006_01_02")))
	if err != nil {
		fmt.Printf("\n\033[31mError creating NDVI video: %s\033[0m\n", err.Error())
		return
	}
	err = output.CreateVideoFromDirectory(imagesPath+"/images/NDRE", fmt.Sprintf("%s/videos/NDRE-%s-%s", imagesPath, startDate.Format("2006_01_02"), endDate.Format("2006_01_02")))
	if err != nil {
		fmt.Printf("\n\033[31mError creating NDRE video: %s\033[0m\n", err.Error())
		return
	}
	err = output.CreateVideoFromDirectory(imagesPath+"/images/NDMI", fmt.Sprintf("%s/videos/NDMI-%s-%s", imagesPath, startDate.Format("2006_01_02"), endDate.Format("2006_01_02")))
	if err != nil {
		fmt.Printf("\n\033[31mError creating NDMI video: %s\033[0m\n", err.Error())
		return
	}

	fmt.Printf("\n\033[32mSuccessfully created videos for NDVI, NDRE, and NDMI indices!\033[0m\n")
}
