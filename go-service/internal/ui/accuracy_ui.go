package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delivery"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/notification"
)

// AccuracyTest handles the UI for testing model accuracy
func AccuracyTest() {
	fmt.Println("\033[33m\nWarning:\033[0m")
	fmt.Println("\033[33mThis will use an existing model as the dataset\033[0m")
	fmt.Println("\033[33mThe model will be split into training and validation portions\033[0m")
	fmt.Println("\033[33mA new training model will be created and tested against validation data\n\033[0m")

	// Select model from available models
	selectedModel, err := SelectModel()
	if err != nil {
		fmt.Printf("\n\033[31m%s\033[0m\n", err.Error())
		return
	}

	fmt.Print("\033[34mEnter the training ratio (percentage, e.g., 80 for 80%%): \033[0m")
	var trainingRatio int
	fmt.Scanln(&trainingRatio)

	if trainingRatio <= 0 || trainingRatio >= 100 {
		fmt.Printf("\n\033[31mInvalid training ratio: %d. Please enter a value between 1 and 99.\033[0m\n", trainingRatio)
		return
	}

	// Create training model filename that preserves the original format
	// Extract the base name without extension
	baseName := strings.TrimSuffix(selectedModel, ".csv")
	// Add training indicator and timestamp
	trainingModelFileName := fmt.Sprintf("%s_training_%s_%d.csv",
		baseName,
		time.Now().Format("2006-01-02"),
		trainingRatio)

	fmt.Printf("\033[32mStarting accuracy test with:\033[0m\n")
	fmt.Printf("\033[32m- Source model: %s\033[0m\n", selectedModel)
	fmt.Printf("\033[32m- Training ratio: %d%%\033[0m\n", trainingRatio)
	fmt.Printf("\033[32m- Training model will be: %s\033[0m\n", trainingModelFileName)

	accuracy, totalTests, correctPredictions, trainingStats, validationStats, accretionMissStats, err := delivery.RunAccuracyTest(
		selectedModel,
		trainingModelFileName,
		trainingRatio,
	)

	if err != nil {
		fmt.Printf("\n\033[31mError during accuracy test: %s\033[0m\n", err.Error())
		if !strings.Contains(err.Error(), "empty csv file given") {
			notification.SendDiscordErrorNotification(fmt.Sprintf("Maxsatt CLI\n\nError during accuracy test: %s", err.Error()))
		}
		return
	}

	accuracyPercentage := accuracy * 100

	fmt.Printf("\n\033[32mAccuracy test completed successfully!\033[0m\n")
	fmt.Printf("\033[32m- Total tests: %d\033[0m\n", totalTests)
	fmt.Printf("\033[32m- Correct predictions: %d\033[0m\n", correctPredictions)
	fmt.Printf("\033[32m- Accuracy: %.2f%%\033[0m\n", accuracyPercentage)

	// Display dataset statistics in console
	fmt.Printf("\n\033[34mDataset Statistics:\033[0m\n")
	fmt.Printf("\033[34mTraining Dataset:\033[0m\n")
	fmt.Printf("\033[34m- Total samples: %d\033[0m\n", trainingStats.TotalSamples)
	fmt.Printf("\033[34mValidation Dataset:\033[0m\n")
	fmt.Printf("\033[34m- Total samples: %d\033[0m\n", validationStats.TotalSamples)

	// Show accretion/miss stats in console
	fmt.Printf("\n\033[35mAccretion/Miss Breakdown:\033[0m\n")
	fmt.Print(delivery.FormatAccretionMissStats(accretionMissStats))

	// Format dataset statistics for Discord message (with percentages)
	trainingStatsFormatted := delivery.FormatDatasetStatsWithPercent(trainingStats, "Training")
	validationStatsFormatted := delivery.FormatDatasetStatsWithPercent(validationStats, "Validation")
	accretionMissFormatted := delivery.FormatAccretionMissStats(accretionMissStats)

	// Create comprehensive Discord message
	discordMessage := fmt.Sprintf("Maxsatt CLI\n\nAccuracy test completed successfully!\n\n"+
		"**Test Results:**\n"+
		"- Source model: %s\n"+
		"- Training ratio: %d%%\n"+
		"- Total tests: %d\n"+
		"- Correct predictions: %d\n"+
		"- Accuracy: %.2f%%\n\n"+
		"%s\n\n%s\n\n%s",
		selectedModel,
		trainingRatio,
		totalTests,
		correctPredictions,
		accuracyPercentage,
		trainingStatsFormatted,
		validationStatsFormatted,
		accretionMissFormatted)

	notification.SendDiscordSuccessNotification(discordMessage)
}
