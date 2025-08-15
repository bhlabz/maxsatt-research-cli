package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delivery"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/notification"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
)

type AccuracyReport struct {
	SourceModel           string
	TrainingModel         string
	TrainingRatio         int
	TestStartTime         time.Time
	TestEndTime           time.Time
	TotalTests            int
	CorrectPredictions    int
	Accuracy              float64
	AccuracyPercentage    float64
	TrainingStats         interface{}
	ValidationStats       interface{}
	AccretionMissStats    interface{}
	TrainingStatsFormatted string
	ValidationStatsFormatted string
	AccretionMissFormatted string
	Error                 string
}

func generateAccuracyMarkdownReport(report *AccuracyReport) error {
	reportPath := fmt.Sprintf("%s/data/reports/accuracy_analysis_%s.md", properties.RootPath(), 
		report.TestStartTime.Format("2006-01-02_15-04-05"))
	
	// Ensure reports directory exists
	reportsDir := fmt.Sprintf("%s/data/reports", properties.RootPath())
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return fmt.Errorf("failed to create reports directory: %w", err)
	}

	file, err := os.Create(reportPath)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	duration := report.TestEndTime.Sub(report.TestStartTime)
	
	content := fmt.Sprintf(`# Model Accuracy Analysis Report

## Test Overview
- **Source Model**: %s
- **Training Model**: %s
- **Training Ratio**: %d%%
- **Test Started**: %s
- **Test Completed**: %s
- **Total Duration**: %s

## Performance Results
- **Total Tests**: %d
- **Correct Predictions**: %d
- **Accuracy**: %.4f (%.2f%%)
- **Error Rate**: %.2f%%

`, report.SourceModel, report.TrainingModel, report.TrainingRatio,
		report.TestStartTime.Format("2006-01-02 15:04:05"),
		report.TestEndTime.Format("2006-01-02 15:04:05"),
		duration.String(),
		report.TotalTests, report.CorrectPredictions, report.Accuracy, report.AccuracyPercentage,
		100 - report.AccuracyPercentage)

	// Add error information if present
	if report.Error != "" {
		content += fmt.Sprintf("## Error Information\n```\n%s\n```\n\n", report.Error)
	}

	// Add training statistics if available
	if report.TrainingStatsFormatted != "" {
		content += "## Training Dataset Statistics\n"
		content += "```\n" + report.TrainingStatsFormatted + "\n```\n\n"
	}

	// Add validation statistics if available
	if report.ValidationStatsFormatted != "" {
		content += "## Validation Dataset Statistics\n"
		content += "```\n" + report.ValidationStatsFormatted + "\n```\n\n"
	}

	// Add accretion/miss breakdown if available
	if report.AccretionMissFormatted != "" {
		content += "## Prediction Accuracy Breakdown\n"
		content += "```\n" + report.AccretionMissFormatted + "\n```\n\n"
	}

	// Add AI analysis recommendations
	content += `## Model Performance Analysis

### Performance Assessment
`
	if report.AccuracyPercentage >= 90 {
		content += "- **Excellent Performance**: Model shows very high accuracy (>90%)\n"
	} else if report.AccuracyPercentage >= 80 {
		content += "- **Good Performance**: Model shows good accuracy (80-90%)\n"
	} else if report.AccuracyPercentage >= 70 {
		content += "- **Moderate Performance**: Model shows moderate accuracy (70-80%)\n"
	} else {
		content += "- **Poor Performance**: Model shows low accuracy (<70%)\n"
	}

	content += `
### AI Analysis Recommendations

#### For Model Improvement:
1. **Feature Engineering**: Review feature selection and engineering techniques
2. **Data Quality**: Analyze training data quality and distribution
3. **Hyperparameter Tuning**: Optimize model parameters for better performance
4. **Cross-Validation**: Implement k-fold cross-validation for robust evaluation

#### For Further Investigation:
1. **Confusion Matrix**: Generate detailed confusion matrix for error analysis
2. **Feature Importance**: Analyze which features contribute most to predictions
3. **Error Analysis**: Investigate patterns in misclassified samples
4. **Data Augmentation**: Consider expanding training dataset if accuracy is low

#### Model Deployment Considerations:
`
	if report.AccuracyPercentage >= 85 {
		content += "- **Ready for Production**: Model performance is suitable for production deployment\n"
	} else if report.AccuracyPercentage >= 75 {
		content += "- **Conditional Deployment**: Model may be suitable with additional validation\n"
	} else {
		content += "- **Not Recommended**: Model requires significant improvement before deployment\n"
	}

	content += fmt.Sprintf(`
## Technical Metadata
- **Evaluation Method**: Train-Validation Split (%d%% training, %d%% validation)
- **Test Type**: Accuracy Assessment
- **Generated on**: %s
- **Model Validation Pipeline**: v1.0

## Statistical Summary
- **Sample Size**: %d test cases
- **Success Rate**: %.2f%%
- **Confidence Level**: Based on %d predictions
- **Model Reliability**: %s

---
*Report generated automatically by Forest Guardian ML Pipeline*
`, report.TrainingRatio, 100-report.TrainingRatio, 
		time.Now().Format("2006-01-02 15:04:05"), 
		report.TotalTests, report.AccuracyPercentage, report.TotalTests,
		func() string {
			if report.AccuracyPercentage >= 90 { return "High" }
			if report.AccuracyPercentage >= 80 { return "Medium-High" }
			if report.AccuracyPercentage >= 70 { return "Medium" }
			return "Low"
		}())

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write report content: %w", err)
	}

	fmt.Printf("Accuracy analysis report generated: %s\n", reportPath)
	return nil
}

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

	// Initialize accuracy report
	report := &AccuracyReport{
		SourceModel:    selectedModel,
		TrainingModel:  trainingModelFileName,
		TrainingRatio:  trainingRatio,
		TestStartTime:  time.Now(),
	}

	accuracy, totalTests, correctPredictions, trainingStats, validationStats, accretionMissStats, err := delivery.RunAccuracyTest(
		selectedModel,
		trainingModelFileName,
		trainingRatio,
	)

	if err != nil {
		fmt.Printf("\n\033[31mError during accuracy test: %s\033[0m\n", err.Error())
		
		// Populate error report
		report.TestEndTime = time.Now()
		report.Error = err.Error()
		
		// Generate error report
		if reportErr := generateAccuracyMarkdownReport(report); reportErr != nil {
			fmt.Printf("Error generating report: %v\n", reportErr)
		}
		return
	}

	accuracyPercentage := accuracy * 100

	// Populate successful test results
	report.TestEndTime = time.Now()
	report.TotalTests = totalTests
	report.CorrectPredictions = correctPredictions
	report.Accuracy = accuracy
	report.AccuracyPercentage = accuracyPercentage
	report.TrainingStats = trainingStats
	report.ValidationStats = validationStats
	report.AccretionMissStats = accretionMissStats

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

	// Format dataset statistics for report
	trainingStatsFormatted := delivery.FormatDatasetStatsWithPercent(trainingStats, "Training")
	validationStatsFormatted := delivery.FormatDatasetStatsWithPercent(validationStats, "Validation")
	accretionMissFormatted := delivery.FormatAccretionMissStats(accretionMissStats)
	
	// Store formatted strings in report
	report.TrainingStatsFormatted = trainingStatsFormatted
	report.ValidationStatsFormatted = validationStatsFormatted
	report.AccretionMissFormatted = accretionMissFormatted

	// Generate comprehensive accuracy analysis report
	if err := generateAccuracyMarkdownReport(report); err != nil {
		fmt.Printf("Error generating accuracy report: %v\n", err)
	}

	// Send notification about test conclusion with accuracy percentage
	conclusionMessage := fmt.Sprintf("Maxsatt CLI\n\nAccuracy test completed successfully!\n\n"+
		"**Test Results:**\n"+
		"- Source model: %s\n"+
		"- Training ratio: %d%%\n"+
		"- Total tests: %d\n"+
		"- Correct predictions: %d\n"+
		"- Accuracy: %.2f%%\n\n"+
		"📊 Detailed analysis report generated in markdown format.",
		selectedModel,
		trainingRatio,
		totalTests,
		correctPredictions,
		accuracyPercentage)
	notification.SendDiscordSuccessNotification(conclusionMessage)
}
