package ui

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delivery"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/notification"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
)

// CreateDataset handles the UI for creating a new dataset
func CreateDataset() {
	fmt.Println("\033[33m\nWarning:\033[0m")
	fmt.Println("\033[33mThe resultant dataset will be created at data/model folder\033[0m")
	fmt.Println("\033[33mThe input data should be a '.csv' file present in data/training_input folder\n\033[0m")

	fmt.Print("\033[34mEnter input data file name: \033[0m")
	var inputDataFileName string
	fmt.Scanln(&inputDataFileName)

	// --- Dataset summary extraction ---
	inputPath := filepath.Join(properties.RootPath(), "data", "training_input", inputDataFileName)
	file, err := os.Open(inputPath)
	if err != nil {
		fmt.Printf("\n\033[31mError opening input file for summary: %s\033[0m\n", err.Error())
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		fmt.Printf("\n\033[31mError reading CSV header: %s\033[0m\n", err.Error())
		return
	}
	colIdx := map[string]int{}
	for i, h := range headers {
		colIdx[h] = i
	}
	// Group by pest and month
	type groupKey struct {
		Pest  string
		Month string
	}
	groupCounts := make(map[groupKey]int)
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		pest, date := "", ""
		if idx, ok := colIdx["pest"]; ok {
			pest = record[idx]
		}
		if idx, ok := colIdx["date"]; ok {
			date = record[idx]
		}
		month := ""
		if len(date) >= 7 {
			month = date[:7]
		}
		key := groupKey{Pest: pest, Month: month}
		groupCounts[key]++
	}
	// Build summary string
	summaryLines := make([]string, 0, len(groupCounts))
	pestToMonths := make(map[string]map[string]struct{})
	for k, count := range groupCounts {
		summaryLines = append(summaryLines, fmt.Sprintf("Pest: %s, Month: %s (%d samples)", k.Pest, k.Month, count))
		if _, ok := pestToMonths[k.Pest]; !ok {
			pestToMonths[k.Pest] = make(map[string]struct{})
		}
		pestToMonths[k.Pest][k.Month] = struct{}{}
	}
	// Add pest-month summary lines
	for pest, months := range pestToMonths {
		monthCount := len(months)
		summaryLines = append(summaryLines, fmt.Sprintf("%d months for pest %s", monthCount, pest))
	}
	summary := strings.Join(summaryLines, "\n")
	// --- End summary extraction ---

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
	err = delivery.CreateDataset(inputDataFileName, outputDataFileName, deltaDays, deltaDaysThreshold, daysBeforeEvidenceToAnalyze)
	if err != nil {
		fmt.Printf("\n\033[31mError creating dataset: %s\033[0m\n", err.Error())
		if !strings.Contains(err.Error(), "empty csv file given") {
			notification.SendDiscordErrorNotification(fmt.Sprintf("Maxsatt CLI\n\nError creating dataset: %s", err.Error()))
		}
		return
	}
	fmt.Printf("\n\033[32mDataset created successfully!\033[0m\n")
	notification.SendDiscordSuccessNotification(fmt.Sprintf("Maxsatt CLI\n\nDataset created successfully! \nFile: %s\n", outputDataFileName))
	// Split summary into chunks if too long for Discord
	const maxDiscordEmbedLen = 1800
	for start := 0; start < len(summary); start += maxDiscordEmbedLen {
		end := start + maxDiscordEmbedLen
		if end > len(summary) {
			end = len(summary)
		}
		chunk := summary[start:end]
		notification.SendDiscordSuccessNotification(fmt.Sprintf("Maxsatt CLI\n\nDataset summary (part):\n%s", chunk))
	}
}
