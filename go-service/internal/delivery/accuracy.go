package delivery

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/dataset"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
	"github.com/gocarina/gocsv"
)

// GroupedData represents data grouped by farm and date
type GroupedData struct {
	Farm string
	Date time.Time
	Rows []ValidationRow
}

// RunAccuracyTest performs the complete accuracy test process
func RunAccuracyTest(sourceModelFileName, trainingModelFileName string, trainingRatio int) (float64, int, int, error) {
	fmt.Println("Starting accuracy test process...")

	// Read and parse the source model dataset
	rows, err := readModelDataset(sourceModelFileName)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to read model dataset: %w", err)
	}

	if len(rows) == 0 {
		return 0, 0, 0, fmt.Errorf("empty model file given")
	}

	// Split data into training and validation sets
	trainingData, validationData := splitModelDataByRatio(rows, trainingRatio)

	fmt.Printf("Split data: %d training rows, %d validation rows\n", len(trainingData), len(validationData))

	// Create training model file with proper naming format
	err = createTrainingModelFile(trainingData, sourceModelFileName, trainingModelFileName)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to create training model: %w", err)
	}

	// Test model accuracy on validation data
	correctPredictions, totalTests, err := testModelAccuracyOnValidation(validationData, trainingModelFileName)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to test model accuracy: %w", err)
	}

	accuracy := float64(correctPredictions) / float64(totalTests)
	return accuracy, totalTests, correctPredictions, nil
}

// readModelDataset reads and parses the model dataset
func readModelDataset(modelFileName string) ([]dataset.FinalData, error) {
	modelDataPath := fmt.Sprintf("%s/data/model/%s", properties.RootPath(), modelFileName)

	file, err := os.OpenFile(modelDataPath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	var rows []dataset.FinalData
	if err := gocsv.UnmarshalFile(file, &rows); err != nil {
		return nil, fmt.Errorf("error unmarshalling CSV: %w", err)
	}

	return rows, nil
}

// splitModelDataByRatio splits the model data into training and validation sets
func splitModelDataByRatio(data []dataset.FinalData, trainingRatio int) ([]dataset.FinalData, []dataset.FinalData) {
	// Shuffle the data for random split
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(data), func(i, j int) {
		data[i], data[j] = data[j], data[i]
	})

	trainingCount := (len(data) * trainingRatio) / 100
	trainingData := data[:trainingCount]
	validationData := data[trainingCount:]

	return trainingData, validationData
}

// createTrainingModelFile creates a training model file from training data
func createTrainingModelFile(trainingData []dataset.FinalData, sourceModelFileName, trainingModelFileName string) error {
	fmt.Println("Creating training model file...")

	filePath := fmt.Sprintf("%s/data/model/%s", properties.RootPath(), trainingModelFileName)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating training model file: %w", err)
	}
	defer file.Close()

	err = gocsv.MarshalFile(&trainingData, file)
	if err != nil {
		return fmt.Errorf("error writing training model file: %w", err)
	}

	fmt.Printf("Training model saved to: %s\n", filePath)
	return nil
}

// testModelAccuracyOnValidation tests the model accuracy on validation data
func testModelAccuracyOnValidation(validationData []dataset.FinalData, trainingModelFileName string) (int, int, error) {
	fmt.Println("Testing model accuracy on validation data...")

	correctPredictions := 0
	totalTests := 0

	// Group validation data by farm and plot for evaluation
	validationGroups := make(map[string][]dataset.FinalData)
	for _, data := range validationData {
		key := fmt.Sprintf("%s_%s", data.Farm, data.Plot)
		validationGroups[key] = append(validationGroups[key], data)
	}

	for _, groupData := range validationGroups {
		if len(groupData) == 0 {
			continue
		}

		// Use the first data point to get farm and plot info
		sample := groupData[0]
		farm := sample.Farm
		plot := sample.Plot

		// Get the most recent date from the group
		var mostRecentDate time.Time
		for _, data := range groupData {
			if data.CreatedAt.After(mostRecentDate) {
				mostRecentDate = data.CreatedAt
			}
		}

		// Evaluate the plot using the trained model
		results, err := EvaluatePlotFinalData(trainingModelFileName, farm, plot, mostRecentDate)
		if err != nil {
			fmt.Printf("Warning: error evaluating %s-%s: %v\n", farm, plot, err)
			continue
		}

		// Check if any result matches the expected label
		for _, result := range results {
			// Find the label with highest probability
			var bestLabel string
			var bestProbability float64

			for _, labelProb := range result.Result {
				if labelProb.Probability > bestProbability {
					bestProbability = labelProb.Probability
					bestLabel = labelProb.Label
				}
			}

			// Find the expected label from the validation data
			var expectedLabel string
			for _, data := range groupData {
				if data.Label != nil {
					expectedLabel = *data.Label
					break
				}
			}

			// Compare with expected label
			if bestLabel == expectedLabel {
				correctPredictions++
			}
			totalTests++
		}
	}

	return correctPredictions, totalTests, nil
}
