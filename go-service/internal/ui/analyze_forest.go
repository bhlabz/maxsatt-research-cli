package ui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delivery"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/notification"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
	"github.com/forest-guardian/forest-guardian-api-poc/output"
)

// AnalyzeForest handles the UI for analyzing pest infestation in forest for a specific date
func AnalyzeForest() {
	fmt.Println("\033[33m\nWarning:\033[0m")
	fmt.Println("\033[33m- A '.geojson' file with the forest name should be present in data/geojsons folder.\033[0m")
	fmt.Println("\033[33m- The '.geojson' file should contain the desired plot in its features identified by plot_id.\n\033[0m")
	reader := bufio.NewReader(os.Stdin)

	modelFolderPath := fmt.Sprintf("%s/data/model/", properties.RootPath())

	modelFiles, err := os.ReadDir(modelFolderPath)
	if err != nil {
		fmt.Printf("\n\033[31mError reading model folder: %s\033[0m\n", err.Error())
		return
	}

	if len(modelFiles) == 0 {
		fmt.Printf("\n\033[31mNo models found in the model folder.\033[0m\n")
		return
	}

	fmt.Println("\033[32m\nAvailable models:\033[0m")
	for i, file := range modelFiles {
		fmt.Printf("\033[32m%d. %s\033[0m\n", i+1, file.Name())
	}

	fmt.Print("\033[34mEnter the number of the model you want to use: \033[0m")
	var modelChoice int
	_, err = fmt.Scan(&modelChoice)
	if err != nil || modelChoice < 1 || modelChoice > len(modelFiles) {
		fmt.Printf("\n\033[31mInvalid choice. Please select a valid model number.\033[0m\n")
		return
	}

	selectedModel := modelFiles[modelChoice-1].Name()
	fmt.Printf("\033[32mYou selected the model: %s\033[0m\n", selectedModel)

	fmt.Print("\033[34mEnter the forest name: \033[0m")
	forest, _ := reader.ReadString('\n')
	forest = strings.TrimSpace(forest)

	endDate, err := ReadDate("Enter the date to be analyzed (YYYY-MM-DD | today): ")
	if err != nil {
		PrintError(err.Error())
		return
	}

	filePath := properties.RootPath() + "/data/geojsons/" + forest + ".geojson"
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("\n\033[31mError opening file: %s\033[0m\n", err.Error())
		return
	}
	defer file.Close()

	var geojsonData map[string]interface{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&geojsonData)
	if err != nil {
		fmt.Printf("\n\033[31mError decoding GEOJSON: %s\033[0m\n", err.Error())
		return
	}

	features, ok := geojsonData["features"].([]interface{})
	if !ok {
		fmt.Printf("\n\033[31mError: Invalid GEOJSON format.\033[0m\n")
		return
	}

	plotIDs := []string{}
	for _, feature := range features {
		featureMap, ok := feature.(map[string]interface{})
		if !ok {
			fmt.Printf("\n\033[31mError: Invalid feature format.\033[0m\n")
			break
		}
		properties, ok := featureMap["properties"].(map[string]interface{})
		if !ok {
			fmt.Printf("\n\033[31mError: Invalid properties format.\033[0m\n")
			break
		}
		plotID, ok := properties["plot_id"].(string)
		if ok {
			plotIDs = append(plotIDs, plotID)
		}
	}
	if len(plotIDs) == 0 {
		fmt.Printf("\n\033[31mNo plot IDs found in the GEOJSON file.\033[0m\n")
		return
	}
	fmt.Printf("\033[33mForest %s has %d plots that will be analyzed\n\033[0m", forest, len(plotIDs))
	errs := []error{}
	startTime := time.Now()
	completed := 0
	for _, plot := range plotIDs {
		fmt.Printf("\033[32m\nStarting forest %s plot %s analysis\033[0m\n", forest, plot)
		fmt.Printf("\033[32m\n- %d/%d \033[0m\n", completed, len(plotIDs))

		result, err := delivery.EvaluatePlotFinalData(selectedModel, forest, plot, endDate)
		if err != nil {
			fmt.Printf("\n\033[31mError evaluating plot: %s\033[0m\n", err.Error())
			if !strings.Contains(err.Error(), "empty csv file given") {
				// notification.SendDiscordErrorNotification(fmt.Sprintf("Maxsatt CLI\n\nError evaluating plot: %s", err.Error()))
			}
			errs = append(errs, err)
			completed++
			continue
		}

		imageFolderPath := fmt.Sprintf("%s/data/images/%s_%s/", properties.RootPath(), forest, plot)

		files, err := os.ReadDir(imageFolderPath)
		if err != nil {
			fmt.Printf("\n\033[31mError reading image folder: %s\033[0m\n", err.Error())
			errs = append(errs, err)
		}

		if len(files) == 0 {
			fmt.Printf("\n\033[31mNo tiff images found to create resultant image\033[0m\n")
			errs = append(errs, err)
			completed++
			continue
		}

		firstFileName := files[0].Name()
		firstFilePath := fmt.Sprintf("%s%s", imageFolderPath, firstFileName)
		resultPath := fmt.Sprintf("%s/data/result/%s/%s/final", properties.RootPath(), forest, plot)

		err = os.MkdirAll(resultPath, os.ModePerm)
		if err != nil {
			fmt.Printf("\n\033[31mFailed to create result folder: %v\033[0m\n", err)
			return
		}

		outputFilePath := fmt.Sprintf("%s/%s_%s_%s_%s", resultPath, forest, plot, endDate.Format("2006-01-02"), strings.TrimSuffix(selectedModel, ".csv"))

		output.CreateFinalDataGeoJson(result, outputFilePath)

		err = output.CreateFinalDataImage(result, firstFilePath, outputFilePath)
		if err != nil {
			fmt.Printf("\n\033[31mError creating resultant image: %s\033[0m\n", err.Error())
			errs = append(errs, err)
			completed++
			continue
		}

		fmt.Printf("\n\033[32mSuccessful analysis!\n Resultant image located at: %s.jpeg\n Resultant geojson located at: %s.geojson\033[0m\n", outputFilePath, outputFilePath)
		completed++
	}
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	if len(errs) > 0 {
		errorMessages := strings.Builder{}
		for _, err := range errs {
			errorMessages.WriteString(fmt.Sprintf("- %s\n", err.Error()))
		}
		notification.SendDiscordErrorNotification(fmt.Sprintf("Maxsatt CLI\n\nErrors occurred during analysis:\n%s", errorMessages.String()))
		fmt.Printf("\n\033[31mErrors occurred during analysis: %s\033[0m\n", errorMessages.String())

	} else {
		notification.SendDiscordSuccessNotification(fmt.Sprintf("Maxsatt CLI\n\nSuccessful forest analysis!\n - Forest: %s\n - Model: %s\n - Date: %s\n - Plots: %d\n - Processing time: %s", forest, selectedModel, endDate.Format("2006-01-02"), len(plotIDs), elapsedTime.String()))
	}
}
