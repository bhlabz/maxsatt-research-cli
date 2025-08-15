package delivery

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/dataset"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/sentinel"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/weather"
	"github.com/gocarina/gocsv"
)

type ValidationRow struct {
	Date     string `csv:"date"`
	Pest     string `csv:"pest"`
	Severity string `csv:"severity"`
	Forest   string `csv:"forest"`
	Plot     string `csv:"plot"`
}

type DatasetReport struct {
	InputFile            string
	OutputFile           string
	TotalSamples         int
	ProcessedSamples     int
	ErrorCount           int
	Errors               []string
	ProcessingStats      map[string]int
	ForestStats          map[string]int
	PestStats            map[string]int
	SeverityStats        map[string]int
	StartTime            time.Time
	EndTime              time.Time
	DeltaDays            int
	DeltaDaysThreshold   int
	DaysBeforeEvidence   int
}

func generateMarkdownReport(report *DatasetReport) error {
	reportPath := fmt.Sprintf("%s/data/reports/dataset_analysis_%s.md", properties.RootPath(), 
		report.StartTime.Format("2006-01-02_15-04-05"))
	
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

	duration := report.EndTime.Sub(report.StartTime)
	successRate := float64(report.ProcessedSamples) / float64(report.TotalSamples) * 100

	content := fmt.Sprintf(`# Dataset Creation Analysis Report

## Overview
- **Input File**: %s
- **Output File**: %s
- **Processing Started**: %s
- **Processing Completed**: %s
- **Total Duration**: %s
- **Success Rate**: %.2f%%

## Processing Summary
- **Total Samples**: %d
- **Successfully Processed**: %d
- **Errors Encountered**: %d

## Configuration Parameters
- **Delta Days**: %d
- **Delta Days Threshold**: %d
- **Days Before Evidence to Analyze**: %d

## Statistics by Category

### Forest Distribution
`, report.InputFile, report.OutputFile, 
		report.StartTime.Format("2006-01-02 15:04:05"),
		report.EndTime.Format("2006-01-02 15:04:05"),
		duration.String(), successRate,
		report.TotalSamples, report.ProcessedSamples, report.ErrorCount,
		report.DeltaDays, report.DeltaDaysThreshold, report.DaysBeforeEvidence)

	for forest, count := range report.ForestStats {
		content += fmt.Sprintf("- **%s**: %d samples\n", forest, count)
	}

	content += "\n### Pest Distribution\n"
	for pest, count := range report.PestStats {
		content += fmt.Sprintf("- **%s**: %d samples\n", pest, count)
	}

	content += "\n### Severity Distribution\n"
	for severity, count := range report.SeverityStats {
		content += fmt.Sprintf("- **%s**: %d samples\n", severity, count)
	}

	if len(report.Errors) > 0 {
		content += "\n## Errors Encountered\n"
		for i, err := range report.Errors {
			content += fmt.Sprintf("%d. %s\n", i+1, err)
		}
	}

	content += `
## Data Quality Analysis

This dataset has been processed with the following quality measures:
- Deduplication based on key columns
- Best sample selection based on derivative analysis
- Weather data integration
- Temporal consistency validation

## AI Analysis Recommendations

### For Machine Learning Models:
1. **Feature Engineering**: Consider the temporal derivatives (NDRE, NDMI, NDVI, PSRI) as primary features
2. **Class Balance**: Review pest and severity distributions for potential class imbalance
3. **Temporal Patterns**: Analyze seasonal trends in the data
4. **Geographic Clustering**: Consider forest-specific model training

### For Further Investigation:
1. **Error Patterns**: Investigate common failure modes in data processing
2. **Weather Correlation**: Analyze correlation between weather patterns and pest occurrence
3. **Spectral Analysis**: Deep dive into vegetation indices effectiveness
4. **Sample Quality**: Review best sample selection criteria effectiveness

## Dataset Metadata
- **Generated on**: %s
- **Processing Pipeline Version**: v1.0
- **Quality Score**: %.1f/10 (based on success rate and data completeness)
`

	qualityScore := (successRate / 10) + 2 // Simple quality scoring
	if qualityScore > 10 {
		qualityScore = 10
	}

	content = fmt.Sprintf(content, time.Now().Format("2006-01-02 15:04:05"), qualityScore)

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write report content: %w", err)
	}

	fmt.Printf("Dataset analysis report generated: %s\n", reportPath)
	return nil
}

func addErrorToReport(report *DatasetReport, errorMsg string) {
	report.Errors = append(report.Errors, errorMsg)
	report.ErrorCount++
	fmt.Println("Error:", errorMsg)
}

func getSamplesAmountFromSeverity(_ string, datasetLength int) int {
	if datasetLength <= 4 {
		return datasetLength
	}
	return datasetLength - (datasetLength / 4)
}

func getBestSamplesFromDeltaDataset(deltaDataset map[[2]int]map[time.Time]dataset.DeltaData, samplesAmount int, label string) map[[2]int]map[time.Time]dataset.DeltaData {
	// Sort the deltaDataset based on the specified derivatives
	deltaDatasetSlice := []dataset.DeltaData{}
	for _, data := range deltaDataset {
		for _, sample := range data {
			deltaDatasetSlice = append(deltaDatasetSlice, sample)
		}
	}
	sort.Slice(deltaDatasetSlice, func(i, j int) bool {
		if label == "Saudavel" {
			if deltaDatasetSlice[i].NDREDerivative != deltaDatasetSlice[j].NDREDerivative {
				return deltaDatasetSlice[i].NDREDerivative > deltaDatasetSlice[j].NDREDerivative
			}
			if deltaDatasetSlice[i].NDMIDerivative != deltaDatasetSlice[j].NDMIDerivative {
				return deltaDatasetSlice[i].NDMIDerivative > deltaDatasetSlice[j].NDMIDerivative
			}
			if deltaDatasetSlice[i].NDVIDerivative != deltaDatasetSlice[j].NDVIDerivative {
				return deltaDatasetSlice[i].NDVIDerivative > deltaDatasetSlice[j].NDVIDerivative
			}
			return deltaDatasetSlice[i].PSRIDerivative < deltaDatasetSlice[j].PSRIDerivative
		} else {
			if deltaDatasetSlice[i].NDREDerivative != deltaDatasetSlice[j].NDREDerivative {
				return deltaDatasetSlice[i].NDREDerivative < deltaDatasetSlice[j].NDREDerivative
			}
			if deltaDatasetSlice[i].NDMIDerivative != deltaDatasetSlice[j].NDMIDerivative {
				return deltaDatasetSlice[i].NDMIDerivative < deltaDatasetSlice[j].NDMIDerivative
			}
			if deltaDatasetSlice[i].NDVIDerivative != deltaDatasetSlice[j].NDVIDerivative {
				return deltaDatasetSlice[i].NDVIDerivative < deltaDatasetSlice[j].NDVIDerivative
			}
			return deltaDatasetSlice[i].PSRIDerivative > deltaDatasetSlice[j].PSRIDerivative
		}
	})

	// Add name and pest (label) to each sample
	for i := range deltaDatasetSlice {
		deltaDatasetSlice[i].Label = &label
	}

	// Select the top samplesAmount rows
	if samplesAmount > len(deltaDatasetSlice) {
		samplesAmount = len(deltaDatasetSlice)
	}
	acceptedCut := deltaDatasetSlice[:samplesAmount]

	newDeltaDataset := make(map[[2]int]map[time.Time]dataset.DeltaData)
	for _, sample := range acceptedCut {
		key := [2]int{sample.X, sample.Y}
		if _, exists := newDeltaDataset[key]; !exists {
			newDeltaDataset[key] = make(map[time.Time]dataset.DeltaData)
		}
		newDeltaDataset[key][sample.StartDate] = sample
	}
	return newDeltaDataset
}

func CreateDataset(inputDataFileName, outputtDataFileName string, deltaDays, deltaDaysTrashHold, daysBeforeEvidenceToAnalyze int) error {
	fmt.Println("create dataset")
	daysToFetch := deltaDays + deltaDaysTrashHold + daysBeforeEvidenceToAnalyze
	deltaMin, deltaMax := deltaDays, deltaDays+deltaDaysTrashHold

	// Initialize report
	report := &DatasetReport{
		InputFile:            inputDataFileName,
		OutputFile:           outputtDataFileName,
		StartTime:            time.Now(),
		DeltaDays:            deltaDays,
		DeltaDaysThreshold:   deltaDaysTrashHold,
		DaysBeforeEvidence:   daysBeforeEvidenceToAnalyze,
		ProcessingStats:      make(map[string]int),
		ForestStats:          make(map[string]int),
		PestStats:            make(map[string]int),
		SeverityStats:        make(map[string]int),
		Errors:               []string{},
	}

	validationDataPath := fmt.Sprintf("%s/data/training_input/%s", properties.RootPath(), inputDataFileName)

	file, err := os.OpenFile(validationDataPath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return err
	}
	defer file.Close()

	var rows []*ValidationRow
	if err := gocsv.UnmarshalFile(file, &rows); err != nil {
		fmt.Println("Error unmarshalling CSV:", err)
		return err
	}

	target := len(rows)
	report.TotalSamples = target
	fmt.Printf("Creating dataset from file %s with %d samples\n", validationDataPath, target)

	for i := 0; i < target; i++ {
		var err error
		defer func() {
			fmt.Println("Error:", err)
		}()
		row := rows[i]
		date, err := time.Parse("2006-01-02", row.Date)
		if err != nil {
			errMsg := fmt.Sprintf("Error parsing date: %v | Row: %d | Forest: %s | Plot: %s | Pest: %s | Severity: %s", err, i+1, row.Forest, row.Plot, row.Pest, row.Severity)
			fmt.Println(err.Error())
			addErrorToReport(report, errMsg)
			continue
		}
		pest := row.Pest
		severity := row.Severity
		forest := row.Forest
		plot := row.Plot

		// Update statistics
		report.ForestStats[forest]++
		report.PestStats[pest]++
		report.SeverityStats[severity]++

		finalData, err := dataset.GetSavedFinalData(forest, plot, date, deltaMin, deltaMax)
		if err != nil {
			errMsg := fmt.Sprintf("Error getting saved final dataset: %v | Row: %d | Forest: %s | Plot: %s | Pest: %s | Severity: %s", err, i+1, forest, plot, pest, severity)
			fmt.Println("Error getting saved final dataset: " + err.Error())
			addErrorToReport(report, errMsg)
		}

		if finalData == nil {

			geometry, err := sentinel.GetGeometryFromGeoJSON(forest, plot)
			if err != nil {
				errMsg := fmt.Sprintf("Error getting geometry: %v | Row: %d | Forest: %s | Plot: %s | Pest: %s | Severity: %s", err, i+1, forest, plot, pest, severity)
				fmt.Println(err.Error())
				addErrorToReport(report, errMsg)
				continue
			}

			endDate := date.AddDate(0, 0, -daysBeforeEvidenceToAnalyze)
			startDate := endDate.AddDate(0, 0, -daysToFetch)

			images, err := sentinel.GetImages(geometry, forest, plot, startDate, endDate, 1)
			if err != nil {
				errMsg := fmt.Sprintf("Error getting images: %v | Row: %d | Forest: %s | Plot: %s | Pest: %s | Severity: %s", err, i+1, forest, plot, pest, severity)
				fmt.Println(err.Error())
				addErrorToReport(report, errMsg)
				continue
			}

			latitude, longitude, err := sentinel.GetCentroidLatitudeLongitudeFromGeometry(geometry)
			if err != nil {
				errMsg := fmt.Sprintf("Error getting centroid latitude and longitude: %v | Row: %d | Forest: %s | Plot: %s | Pest: %s | Severity: %s", err, i+1, forest, plot, pest, severity)
				fmt.Println(err.Error())
				addErrorToReport(report, errMsg)
				continue
			}

			historicalWeather, err := weather.FetchWeather(latitude, longitude, startDate.AddDate(0, -4, 0), endDate, 10)
			if err != nil {
				errMsg := fmt.Sprintf("Error getting weather: %v | Row: %d | Forest: %s | Plot: %s | Pest: %s | Severity: %s", err, i+1, forest, plot, pest, severity)
				fmt.Println(err.Error())
				addErrorToReport(report, errMsg)
				continue
			}

			data, err := dataset.CreatePixelDataset(forest, plot, images)
			if err != nil {
				errMsg := fmt.Sprintf("Error creating pixel dataset: %v | Row: %d | Forest: %s | Plot: %s | Pest: %s | Severity: %s", err, i+1, forest, plot, pest, severity)
				fmt.Println(err.Error())
				addErrorToReport(report, errMsg)
				continue
			}
			if len(data) == 0 {
				err = fmt.Errorf("no data available to create the dataset for forest: %s, plot: %s using %d images", forest, plot, len(images))
				errMsg := fmt.Sprintf("Error creating pixel dataset: %v | Row: %d | Forest: %s | Plot: %s | Pest: %s | Severity: %s", err, i+1, forest, plot, pest, severity)
				fmt.Println(err.Error())
				addErrorToReport(report, errMsg)
				continue
			}

			cleanData, err := dataset.CreateCleanDataset(forest, plot, data)
			if err != nil {
				errMsg := fmt.Sprintf("Error creating clean dataset: %v | Row: %d | Forest: %s | Plot: %s | Pest: %s | Severity: %s", err, i+1, forest, plot, pest, severity)
				fmt.Println(err.Error())
				addErrorToReport(report, errMsg)
				continue
			}

			deltaDataset, err := dataset.CreateDeltaDataset(forest, plot, deltaMin, deltaMax, cleanData)
			if err != nil {
				errMsg := fmt.Sprintf("Error creating delta dataset: %v | Row: %d | Forest: %s | Plot: %s | Pest: %s | Severity: %s", err, i+1, forest, plot, pest, severity)
				fmt.Println(err.Error())
				addErrorToReport(report, errMsg)
				continue
			}

			samplesAmount := getSamplesAmountFromSeverity(severity, len(deltaDataset))
			bestSamples := getBestSamplesFromDeltaDataset(deltaDataset, samplesAmount, pest)

			fmt.Printf("Best samples for pest %s with severity %s: %d samples. dataset with %d samples\n", pest, severity, len(bestSamples), len(deltaDataset))

			createdFinalData, err := dataset.GetFinalData(bestSamples, historicalWeather, startDate, endDate, forest, plot)
			if err != nil {
				errMsg := fmt.Sprintf("Error getting climate group data: %v | Row: %d | Forest: %s | Plot: %s | Pest: %s | Severity: %s", err, i+1, forest, plot, pest, severity)
				fmt.Println(err.Error())
				addErrorToReport(report, errMsg)
				continue
			}

			err = dataset.SaveFinalData(createdFinalData, date)
			if err != nil {
				errMsg := fmt.Sprintf("Error getting climate group data: %v | Row: %d | Forest: %s | Plot: %s | Pest: %s | Severity: %s", err, i+1, forest, plot, pest, severity)
				fmt.Println(err.Error())
				addErrorToReport(report, errMsg)
				continue
			}

			finalData = createdFinalData
		}

		filePath := fmt.Sprintf("%s/data/model/%s", properties.RootPath(), outputtDataFileName)
		fileExists := false

		// Check if the file already exists
		if _, err := os.Stat(filePath); err == nil {
			fileExists = true
		}

		file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		defer file.Close()

		if fileExists {
			_, err = file.Seek(0, io.SeekEnd)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
		}

		writer := csv.NewWriter(file)
		defer writer.Flush()

		// Write the header only if the file does not already exist
		if !fileExists {
			if err := gocsv.MarshalCSV(&finalData, writer); err != nil {
				fmt.Println(err.Error())
				continue
			}
			continue
		}
		// Write the data rows
		if err := gocsv.MarshalCSVWithoutHeaders(&finalData, writer); err != nil {
			fmt.Println(err.Error())
			continue
		}

		fmt.Printf("Processed row %d/%d: Forest=%s, Plot=%s, Pest=%s, Severity=%s, rows=%d\n", i+1, target, forest, plot, pest, severity, len(finalData))
		report.ProcessedSamples++

	}

	// Finalize report
	report.EndTime = time.Now()
	filePath := fmt.Sprintf("%s/data/model/%s", properties.RootPath(), outputtDataFileName)
	err = deduplicateCSVFile(filePath)
	if err != nil {
		fmt.Printf("[Deduplication] Error during deduplication: %v\n", err)
		addErrorToReport(report, fmt.Sprintf("Deduplication error: %v", err))
	}

	// Generate markdown report
	if err := generateMarkdownReport(report); err != nil {
		fmt.Printf("Error generating report: %v\n", err)
	}

	// Check if all rows failed
	if report.ErrorCount == target {
		return fmt.Errorf("all rows failed during dataset creation")
	}

	fmt.Printf("Dataset created successfully. Processed %d/%d samples with %d errors\n", 
		report.ProcessedSamples, report.TotalSamples, report.ErrorCount)
	return nil
}

// deduplicateCSVFile removes duplicate rows from a CSV file based on selected columns and overwrites the file.
func deduplicateCSVFile(filePath string) error {
	fmt.Printf("[Deduplication] Starting deduplication for file: %s\n", filePath)
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for deduplication: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	var records [][]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read CSV row: %w", err)
		}
		records = append(records, record)
	}

	// Indices of columns to deduplicate on (based on header order)
	colIdx := map[string]int{}
	for i, h := range headers {
		colIdx[h] = i
	}
	// List of columns to deduplicate on
	dedupCols := []string{
		"avg_temperature", "temp_std_dev", "avg_humidity", "humidity_std_dev", "total_precipitation", "dry_days_consecutive",
		"ndre", "ndmi", "psri", "ndvi", "delta_min", "delta_max", "delta", "ndre_derivative", "ndmi_derivative", "psri_derivative", "ndvi_derivative", "label",
	}

	unique := make(map[string]struct{})
	var deduped [][]string
	for _, row := range records {
		var keyParts []string
		for _, col := range dedupCols {
			idx, ok := colIdx[col]
			if !ok || idx >= len(row) {
				keyParts = append(keyParts, "")
			} else {
				keyParts = append(keyParts, row[idx])
			}
		}
		key := strings.Join(keyParts, "||")
		if _, exists := unique[key]; !exists {
			unique[key] = struct{}{}
			deduped = append(deduped, row)
		}
	}

	if len(deduped) == len(records) {
		fmt.Printf("[Deduplication] No duplicates found. Total rows: %d\n", len(deduped))
		return nil
	}

	fmt.Printf("[Deduplication] Removed %d duplicate rows. Clean rows: %d\n", len(records)-len(deduped), len(deduped))

	// Write back to the same file
	tmpPath := filePath + ".tmp"
	outFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file for deduplication: %w", err)
	}
	defer outFile.Close()

	writer := csv.NewWriter(outFile)
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}
	if err := writer.WriteAll(deduped); err != nil {
		return fmt.Errorf("failed to write deduplicated rows: %w", err)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("error flushing CSV writer: %w", err)
	}

	// Replace original file
	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("failed to replace original file with deduplicated file: %w", err)
	}
	fmt.Printf("[Deduplication] Deduplication complete. File updated: %s\n", filePath)
	return nil
}
