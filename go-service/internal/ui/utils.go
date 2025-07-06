package ui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
)

// Colors for consistent UI
const (
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorReset  = "\033[0m"
)

// PrintWarning displays a warning message with consistent formatting
func PrintWarning(message string) {
	fmt.Printf("%s\nWarning:%s\n", ColorYellow, ColorReset)
	fmt.Printf("%s%s%s\n", ColorYellow, message, ColorReset)
}

// PrintError displays an error message with consistent formatting
func PrintError(message string) {
	fmt.Printf("\n%sError: %s%s\n", ColorRed, message, ColorReset)
}

// PrintSuccess displays a success message with consistent formatting
func PrintSuccess(message string) {
	fmt.Printf("\n%s%s%s\n", ColorGreen, message, ColorReset)
}

// PrintInfo displays an info message with consistent formatting
func PrintInfo(message string) {
	fmt.Printf("%s%s%s", ColorBlue, message, ColorReset)
}

// ReadString reads a string from stdin with trimming
func ReadString(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	PrintInfo(prompt)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// ReadInt reads an integer from stdin with validation
func ReadInt(prompt string, min, max int) (int, error) {
	PrintInfo(prompt)
	var input string
	fmt.Scanln(&input)
	input = strings.TrimSpace(input)

	value, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", input)
	}

	if value < min || value > max {
		return 0, fmt.Errorf("value must be between %d and %d", min, max)
	}

	return value, nil
}

// ReadDate reads a date from stdin with validation
func ReadDate(prompt string) (time.Time, error) {
	input := ReadString(prompt)
	if input == "today" {
		return time.Now(), nil
	}
	date, err := time.Parse("2006-01-02", input)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format: %s. Please use YYYY-MM-DD", input)
	}
	return date, nil
}

// ReadPositiveInt reads a positive integer from stdin
func ReadPositiveInt(prompt string) (int, error) {
	PrintInfo(prompt)
	var input string
	fmt.Scanln(&input)
	input = strings.TrimSpace(input)

	value, err := strconv.Atoi(input)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid number: %s. Please enter a positive integer", input)
	}

	return value, nil
}

// SelectModel displays available models and returns the selected one
func SelectModel() (string, error) {
	modelFolderPath := fmt.Sprintf("%s/data/model/", properties.RootPath())

	modelFiles, err := os.ReadDir(modelFolderPath)
	if err != nil {
		return "", fmt.Errorf("error reading model folder: %s", err.Error())
	}

	if len(modelFiles) == 0 {
		return "", fmt.Errorf("no models found in the model folder")
	}

	fmt.Printf("%s\nAvailable models:%s\n", ColorGreen, ColorReset)
	for i, file := range modelFiles {
		fmt.Printf("%s%d. %s%s\n", ColorGreen, i+1, file.Name(), ColorReset)
	}

	choice, err := ReadInt("Enter the number of the model you want to use: ", 1, len(modelFiles))
	if err != nil {
		return "", err
	}

	selectedModel := modelFiles[choice-1].Name()
	fmt.Printf("%sYou selected the model: %s%s\n", ColorGreen, selectedModel, ColorReset)

	return selectedModel, nil
}

// ReadForestAndPlot reads forest and plot information
func ReadForestAndPlot() (string, string, error) {
	PrintInfo("Available forests: ")
	ListForests()
	forest := ReadString("Enter the forest name: ")
	PrintInfo("Available plots: ")
	ListPlots(forest)
	plot := ReadString("Enter the plot id: ")

	if forest == "" || plot == "" {
		return "", "", fmt.Errorf("forest name and plot id cannot be empty")
	}

	return forest, plot, nil
}

// ReadDateRange reads end date and number of days to calculate start date
func ReadDateRange() (time.Time, time.Time, error) {
	endDate, err := ReadDate("Enter the end date (YYYY-MM-DD): ")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	days, err := ReadPositiveInt("Enter number of days: ")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	startDate := endDate.AddDate(0, 0, -days)
	return startDate, endDate, nil
}

// GetPlotIDsFromGeoJSON reads plot IDs from a GeoJSON file
func GetPlotIDsFromGeoJSON(forest string) ([]string, error) {
	filePath := properties.RootPath() + "/data/geojsons/" + forest + ".geojson"
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %s", err.Error())
	}
	defer file.Close()

	var geojsonData map[string]interface{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&geojsonData)
	if err != nil {
		return nil, fmt.Errorf("error decoding GEOJSON: %s", err.Error())
	}

	features, ok := geojsonData["features"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid GEOJSON format")
	}

	plotIDs := []string{}
	for _, feature := range features {
		featureMap, ok := feature.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid feature format")
		}
		properties, ok := featureMap["properties"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid properties format")
		}
		plotID, ok := properties["plot_id"].(string)
		if ok {
			plotIDs = append(plotIDs, plotID)
		}
	}

	if len(plotIDs) == 0 {
		return nil, fmt.Errorf("no plot IDs found in the GEOJSON file")
	}

	return plotIDs, nil
}

// ReadPixelCoordinates reads pixel coordinates from user input
func ReadPixelCoordinates() ([]struct{ X, Y int }, error) {
	var pixels []struct{ X, Y int }

	for {
		input := ReadString("Enter pixel coordinates (x,y) or 'done' to finish: ")
		if strings.ToLower(input) == "done" {
			break
		}

		parts := strings.Split(input, ",")
		if len(parts) != 2 {
			PrintError("Invalid format. Please use x,y")
			continue
		}

		x, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			PrintError("Invalid x coordinate")
			continue
		}

		y, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			PrintError("Invalid y coordinate")
			continue
		}

		pixels = append(pixels, struct{ X, Y int }{X: x, Y: y})
	}

	return pixels, nil
}

// CreateResultDirectory creates the result directory structure
func CreateResultDirectory(forest, plot, resultType string) (string, error) {
	resultPath := fmt.Sprintf("%s/data/result/%s/%s/%s", properties.RootPath(), forest, plot, resultType)
	err := os.MkdirAll(resultPath, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to create result folder: %v", err)
	}
	return resultPath, nil
}
