package delivery

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/dataset"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/notification"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
	"github.com/gocarina/gocsv"
)

// GroupedData represents data grouped by forest and date
type GroupedData struct {
	Forest string
	Date   time.Time
	Rows   []ValidationRow
}

// DatasetStats represents statistics about a dataset
type DatasetStats struct {
	TotalSamples          int
	PestMonthDistribution map[string]int // key: pest|YYYY-MM
}

// Add new struct for detailed stats

type AccretionMissStats struct {
	ForestPlot map[string]map[string]map[string]*PestStats // forest -> plot -> pest -> stats
	TotalTests int
}

type PestStats struct {
	Accretions   int
	Misses       int
	MissAffirmed map[string]int // pest label that was affirmed instead
}

// RunAccuracyTest performs the complete accuracy test process
func RunAccuracyTest(sourceModelFileName, trainingModelFileName string, trainingRatio int) (float64, int, int, *DatasetStats, *DatasetStats, *AccretionMissStats, error) {
	fmt.Println("Starting accuracy test process...")

	// Read and parse the source model dataset
	rows, err := readModelDataset(sourceModelFileName)
	if err != nil {
		return 0, 0, 0, nil, nil, nil, fmt.Errorf("failed to read model dataset: %w", err)
	}

	if len(rows) == 0 {
		return 0, 0, 0, nil, nil, nil, fmt.Errorf("empty model file given")
	}

	// Split data into training and validation sets
	trainingData, validationData := splitModelDataByRatio(rows, trainingRatio)

	// Calculate training and validation dataset statistics
	trainingStats := calculateDatasetStats(trainingData)
	validationStats := calculateDatasetStats(validationData)

	fmt.Printf("Split data: %d training rows, %d validation rows\n", len(trainingData), len(validationData))

	// Create training model file with proper naming format
	err = createTrainingModelFile(trainingData, sourceModelFileName, trainingModelFileName)
	if err != nil {
		return 0, 0, 0, nil, nil, nil, fmt.Errorf("failed to create training model: %w", err)
	}

	// Test model accuracy on validation data
	correctPredictions, totalTests, accretionMissStats, err := testModelAccuracyOnValidation(validationData, trainingModelFileName)
	if err != nil {
		return 0, 0, 0, nil, nil, nil, fmt.Errorf("failed to test model accuracy: %w", err)
	}
	accuracy := float64(correctPredictions) / float64(totalTests)

	// Clean up the training model file after testing
	cleanupTrainingModelFile(trainingModelFileName)

	return accuracy, totalTests, correctPredictions, trainingStats, validationStats, accretionMissStats, nil
}

// cleanupTrainingModelFile removes the training model file after accuracy testing
func cleanupTrainingModelFile(trainingModelFileName string) {
	filePath := fmt.Sprintf("%s/data/model/%s", properties.RootPath(), trainingModelFileName)
	err := os.Remove(filePath)
	if err != nil {
		fmt.Printf("Warning: Failed to delete training model file %s: %v\n", trainingModelFileName, err)
	} else {
		fmt.Printf("Training model file %s deleted successfully\n", trainingModelFileName)
	}
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

// splitModelDataByRatio splits the model data into training and validation sets by group (EndDate, Label, Forest, Plot)
func splitModelDataByRatio(data []dataset.FinalData, trainingRatio int) ([]dataset.FinalData, []dataset.FinalData) {
	type groupKey struct {
		EndDate time.Time
		Label   string
		Forest  string
		Plot    string
	}

	// Group data by (EndDate, Label, Forest, Plot)
	groups := make(map[groupKey][]dataset.FinalData)
	for _, row := range data {
		label := ""
		if row.Label != nil {
			label = *row.Label
		}
		key := groupKey{
			EndDate: row.EndDate,
			Label:   label,
			Forest:  row.Forest,
			Plot:    row.Plot,
		}
		groups[key] = append(groups[key], row)
	}

	// Collect all group keys
	var keys []groupKey
	for k := range groups {
		keys = append(keys, k)
	}

	// Shuffle the group keys
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(keys), func(i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
	})

	// Split groups by ratio
	trainingGroupCount := (len(keys) * trainingRatio) / 100
	var trainingData, validationData []dataset.FinalData
	for i, k := range keys {
		if i < trainingGroupCount {
			trainingData = append(trainingData, groups[k]...)
		} else {
			validationData = append(validationData, groups[k]...)
		}
	}

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
func testModelAccuracyOnValidation(validationData []dataset.FinalData, trainingModelFileName string) (int, int, *AccretionMissStats, error) {
	fmt.Println("Testing model accuracy on validation data...")

	correctPredictions := 0
	totalTests := 0

	stats := &AccretionMissStats{ForestPlot: make(map[string]map[string]map[string]*PestStats)}

	// Group validation data by forest and plot for evaluation
	validationGroups := make(map[string][]dataset.FinalData)
	for _, data := range validationData {
		key := fmt.Sprintf("%s_%s", data.Forest, data.Plot)
		validationGroups[key] = append(validationGroups[key], data)
	}

	for _, groupData := range validationGroups {
		if len(groupData) == 0 {
			continue
		}

		// Use the first data point to get forest and plot info
		sample := groupData[0]
		forest := sample.Forest
		plot := sample.Plot

		// Get the most recent date from the group
		var mostRecentDate time.Time
		for _, data := range groupData {
			if data.CreatedAt.After(mostRecentDate) {
				mostRecentDate = data.CreatedAt
			}
		}

		// Evaluate the plot using the trained model
		results, err := EvaluatePlotFinalData(trainingModelFileName, forest, plot, mostRecentDate)
		if err != nil {
			fmt.Printf("Warning: error evaluating %s-%s: %v\n", forest, plot, err)
			notification.SendDiscordWarnNotification(fmt.Sprintf("Warning: error evaluating %s-%s: %v\n", forest, plot, err))
			continue
		}

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

			if _, ok := stats.ForestPlot[forest]; !ok {
				stats.ForestPlot[forest] = make(map[string]map[string]*PestStats)
			}
			if _, ok := stats.ForestPlot[forest][plot]; !ok {
				stats.ForestPlot[forest][plot] = make(map[string]*PestStats)
			}
			if _, ok := stats.ForestPlot[forest][plot][expectedLabel]; !ok {
				stats.ForestPlot[forest][plot][expectedLabel] = &PestStats{MissAffirmed: make(map[string]int)}
			}

			if bestLabel == expectedLabel {
				correctPredictions++
				stats.ForestPlot[forest][plot][expectedLabel].Accretions++
			} else {
				stats.ForestPlot[forest][plot][expectedLabel].Misses++
				stats.ForestPlot[forest][plot][expectedLabel].MissAffirmed[bestLabel]++
			}
			totalTests++
		}
	}
	stats.TotalTests = totalTests
	return correctPredictions, totalTests, stats, nil
}

// calculateDatasetStats calculates comprehensive statistics for a dataset
func calculateDatasetStats(data []dataset.FinalData) *DatasetStats {
	stats := &DatasetStats{
		TotalSamples:          len(data),
		PestMonthDistribution: make(map[string]int),
	}
	for _, row := range data {
		label := "unknown"
		if row.Label != nil && *row.Label != "" {
			label = *row.Label
		}
		month := row.EndDate.Format("2006-01")
		key := label + "|" + month
		stats.PestMonthDistribution[key]++
	}
	return stats
}

// FormatDatasetStats formats dataset statistics for Discord message
func FormatDatasetStats(stats *DatasetStats, datasetName string) string {
	if stats == nil {
		return fmt.Sprintf("%s: No data available", datasetName)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**%s Dataset:**\n", datasetName))
	sb.WriteString(fmt.Sprintf("- Total samples: %d\n", stats.TotalSamples))

	// Pest/Month distribution
	if len(stats.PestMonthDistribution) > 0 {
		sb.WriteString("- Pest/Month distribution:\n")
		// Collect and sort keys for stable output
		type pestMonthCount struct {
			pest  string
			month string
			count int
		}
		var counts []pestMonthCount
		for key, count := range stats.PestMonthDistribution {
			parts := strings.SplitN(key, "|", 2)
			pest, month := parts[0], "unknown"
			if len(parts) > 1 {
				month = parts[1]
			}
			counts = append(counts, pestMonthCount{pest, month, count})
		}
		sort.Slice(counts, func(i, j int) bool {
			if counts[i].pest == counts[j].pest {
				return counts[i].month < counts[j].month
			}
			return counts[i].pest < counts[j].pest
		})
		for _, c := range counts {
			sb.WriteString(fmt.Sprintf("  • %s (%s): %d samples\n", c.pest, c.month, c.count))
		}
	}

	return sb.String()
}

// Add formatting function for accretion/miss stats
func FormatAccretionMissStats(stats *AccretionMissStats) string {
	if stats == nil || stats.TotalTests == 0 {
		return "No accretion/miss data available."
	}
	var sb strings.Builder
	for forest, plots := range stats.ForestPlot {
		for plot, pests := range plots {
			for pest, pestStats := range pests {
				total := pestStats.Accretions + pestStats.Misses
				if total == 0 {
					continue
				}
				accretionPct := float64(pestStats.Accretions) / float64(total) * 100
				if pestStats.Accretions > 0 {
					sb.WriteString(fmt.Sprintf("Forest %s plot %s had %.1f%% accretions on pest %s\n", forest, plot, accretionPct, pest))
				}
				if pestStats.Misses > 0 {
					for affirmed, count := range pestStats.MissAffirmed {
						affirmPct := float64(count) / float64(total) * 100
						sb.WriteString(fmt.Sprintf("Forest %s plot %s had %.1f%% misses on pest %s affirming pest %s\n", forest, plot, affirmPct, pest, affirmed))
					}
				}
			}
		}
	}
	return sb.String()
}

// Add a new function to format dataset stats with percentages
func FormatDatasetStatsWithPercent(stats *DatasetStats, datasetName string) string {
	if stats == nil || stats.TotalSamples == 0 {
		return fmt.Sprintf("%s: No data available", datasetName)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**%s Dataset:**\n", datasetName))
	sb.WriteString(fmt.Sprintf("- Total samples: %d\n", stats.TotalSamples))
	if len(stats.PestMonthDistribution) > 0 {
		sb.WriteString("- Pest/Month distribution (percentages):\n")
		type pestMonthCount struct {
			pest  string
			month string
			count int
		}
		var counts []pestMonthCount
		for key, count := range stats.PestMonthDistribution {
			parts := strings.SplitN(key, "|", 2)
			pest, month := parts[0], "unknown"
			if len(parts) > 1 {
				month = parts[1]
			}
			counts = append(counts, pestMonthCount{pest, month, count})
		}
		sort.Slice(counts, func(i, j int) bool {
			if counts[i].pest == counts[j].pest {
				return counts[i].month < counts[j].month
			}
			return counts[i].pest < counts[j].pest
		})
		for _, c := range counts {
			pct := float64(c.count) / float64(stats.TotalSamples) * 100
			sb.WriteString(fmt.Sprintf("  • %s (%s): %d samples (%.1f%%)\n", c.pest, c.month, c.count, pct))
		}
	}
	return sb.String()
}
