package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delivery"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
	"github.com/forest-guardian/forest-guardian-api-poc/output"
)

// AnalyzePlot handles the UI for analyzing pest infestation in a forest plot for a specific date
func AnalyzePlot() {
	PrintWarning("- A '.geojson' file with the forest name should be present in data/geojsons folder.\n- The '.geojson' file should contain the desired plot in its features identified by plot_id.")

	// Select model
	selectedModel, err := SelectModel()
	if err != nil {
		PrintError(err.Error())
		return
	}

	// Read forest and plot
	forest, plot, err := ReadForestAndPlot()
	if err != nil {
		PrintError(err.Error())
		return
	}

	// Read date
	endDate, err := ReadDate("Enter the date to be analyzed (YYYY-MM-DD | today): ")
	if err != nil {
		PrintError(err.Error())
		return
	}

	// Evaluate plot
	result, err := delivery.EvaluatePlotFinalData(selectedModel, forest, plot, endDate)
	if err != nil {
		PrintError(fmt.Sprintf("Error evaluating plot: %s", err.Error()))
		return
	}

	// Check for images
	imageFolderPath := fmt.Sprintf("%s/data/images/%s_%s/", properties.RootPath(), forest, plot)
	files, err := os.ReadDir(imageFolderPath)
	if err != nil {
		PrintError(fmt.Sprintf("Error reading image folder: %s", err.Error()))
		return
	}

	if len(files) == 0 {
		PrintError("No tiff images found to create resultant image")
		return
	}

	// Create result directory
	resultPath, err := CreateResultDirectory(forest, plot, "final")
	if err != nil {
		PrintError(err.Error())
		return
	}

	// Create output files
	firstFileName := files[0].Name()
	firstFilePath := fmt.Sprintf("%s%s", imageFolderPath, firstFileName)
	outputFilePath := fmt.Sprintf("%s/%s_%s_%s_%s", resultPath, forest, plot, endDate.Format("2006-01-02"), strings.TrimSuffix(selectedModel, ".csv"))

	output.CreateFinalDataGeoJson(result, outputFilePath)

	err = output.CreateFinalDataImage(result, firstFilePath, outputFilePath)
	if err != nil {
		PrintError(fmt.Sprintf("Error creating resultant image: %s", err.Error()))
		return
	}

	PrintSuccess(fmt.Sprintf("Successful analysis!\nResultant image located at: %s.jpeg\nResultant geojson located at: %s.geojson", outputFilePath, outputFilePath))
}
