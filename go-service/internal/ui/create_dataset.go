package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delivery"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/notification"
)

// CreateDataset handles the UI for creating a new dataset
func CreateDataset() {
	fmt.Println("\033[33m\nWarning:\033[0m")
	fmt.Println("\033[33mThe resultant dataset will be created at data/model folder\033[0m")
	fmt.Println("\033[33mThe input data should be a '.csv' file present in data/training_input folder\n\033[0m")

	fmt.Print("\033[34mEnter input data file name: \033[0m")
	var inputDataFileName string
	fmt.Scanln(&inputDataFileName)

	fmt.Print("\033[34mEnter the ideal delta days for the image analysis: \033[0m")
	var deltaDays int
	fmt.Scanln(&deltaDays)

	fmt.Print("\033[34mEnter the delta days trash hold for the image analysis: \033[0m")
	var deltaDaysThreshold int
	fmt.Scanln(&deltaDaysThreshold)

	fmt.Print("\033[34mEnter the days before evidence to analyze: \033[0m")
	var daysBeforeEvidenceToAnalyze int
	fmt.Scanln(&daysBeforeEvidenceToAnalyze)

	outputDataFileName := fmt.Sprintf("%s_%s_%d_%d_%d.csv", strings.TrimSuffix(inputDataFileName, ".csv"), time.Now().Format("2006-01-02"), deltaDays, deltaDaysThreshold, daysBeforeEvidenceToAnalyze)
	err := delivery.CreateDataset(inputDataFileName, outputDataFileName, deltaDays, deltaDaysThreshold, daysBeforeEvidenceToAnalyze)
	if err != nil {
		fmt.Printf("\n\033[31mError creating dataset: %s\033[0m\n", err.Error())
		if !strings.Contains(err.Error(), "empty csv file given") {
			notification.SendDiscordErrorNotification(fmt.Sprintf("Maxsatt CLI\n\nError creating dataset: %s", err.Error()))
		}
		return
	}
	fmt.Printf("\n\033[32mDataset created successfully!\033[0m\n")
	notification.SendDiscordSuccessNotification(fmt.Sprintf("Maxsatt CLI\n\nDataset created successfully! \n\nFile: %s", outputDataFileName))
}
