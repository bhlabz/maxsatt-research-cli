package spread

import (
	"fmt"
	"sort"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delta"
)

type PestSpreadSample struct {
	delta.Data
	Cluster  int
	Severity int
}

func PestSpread(deltaData map[[2]int]map[time.Time]delta.Data, daysToCluster int) (map[time.Time][]PestSpreadSample, error) {
	groupedData := make(map[time.Time][]delta.Data)
	for _, datePixel := range deltaData {
		for date, pixel := range datePixel {
			if _, exists := groupedData[date]; !exists {
				groupedData[date] = []delta.Data{}
			}
			groupedData[date] = append(groupedData[date], pixel)
		}
	}

	// Create gRPC client for pest clustering
	client, err := NewPestClusteringClient("localhost:50051")
	if err != nil {
		return nil, fmt.Errorf("failed to create pest clustering client: %w", err)
	}
	defer client.Close()

	var allPestSpreadSamples = make(map[time.Time][]PestSpreadSample)

	// Get dates and sort them
	dates := make([]time.Time, 0, len(groupedData))
	for d := range groupedData {
		dates = append(dates, d)
	}
	sort.Slice(dates, func(i, j int) bool {
		return dates[i].Before(dates[j])
	})

	// Process in batches
	for i := 0; i < len(dates); i += daysToCluster {
		end := i + daysToCluster
		if end > len(dates) {
			end = len(dates)
		}
		batchDates := dates[i:end]

		var batchPixels []delta.Data
		for _, date := range batchDates {
			batchPixels = append(batchPixels, groupedData[date]...)
		}

		if len(batchPixels) == 0 {
			continue
		}

		fmt.Println("Processing pixels for dates:", batchDates[0].Format("2006-01-02"), "to", batchDates[len(batchDates)-1].Format("2006-01-02"))
		fmt.Printf("Sending %d pixels to clustering service", len(batchPixels))

		// Call the clustering function for each group of pixels
		pestSpreadSamples, err := client.ClusterizeSpread(batchPixels)
		if err != nil {
			fmt.Printf("Error clustering pixels for dates %v: %v", batchDates, err)
			continue
		}

		fmt.Printf("Received %d clustered samples for dates %v", len(pestSpreadSamples), batchDates)

		// Log cluster information
		clusterCounts := make(map[int]int)
		for _, sample := range pestSpreadSamples {
			clusterCounts[sample.Cluster]++
		}

		fmt.Printf("Cluster distribution for dates %v:", batchDates)
		for cluster, count := range clusterCounts {
			fmt.Printf("  Cluster %d: %d samples", cluster, count)
		}

		// Regroup clustered samples by date
		for _, sample := range pestSpreadSamples {
			dateKey := sample.Data.EndDate
			if _, exists := allPestSpreadSamples[dateKey]; !exists {
				allPestSpreadSamples[dateKey] = []PestSpreadSample{}
			}
			allPestSpreadSamples[dateKey] = append(allPestSpreadSamples[dateKey], sample)
		}
	}

	fmt.Printf("Total pest spread samples processed: %d", len(allPestSpreadSamples))

	for date, data := range allPestSpreadSamples {
		severityTable := findClusterSeverityDescSeveritySortingTable(data)
		for i, value := range data {
			allPestSpreadSamples[date][i].Severity = severityTable[value.Cluster]
		}
	}

	return allPestSpreadSamples, nil
}

func findClusterSeverityDescSeveritySortingTable(data []PestSpreadSample) map[int]int {
	clusterMeans := make(map[int]float64)
	clusterCounts := make(map[int]int)

	for _, sample := range data {
		sortValue := sample.NDRE + sample.NDMI + (sample.PSRI * -1) + sample.NDVI + sample.NDREDerivative + sample.NDMIDerivative + (sample.PSRIDerivative * -1) + sample.NDVIDerivative
		clusterMeans[sample.Cluster] += sortValue
		clusterCounts[sample.Cluster]++
	}

	for cluster, total := range clusterMeans {
		clusterMeans[cluster] = total / float64(clusterCounts[cluster])
	}

	// Create a slice of cluster-mean pairs for sorting
	type clusterMean struct {
		cluster int
		mean    float64
	}

	var clusterMeansList []clusterMean
	for cluster, mean := range clusterMeans {
		clusterMeansList = append(clusterMeansList, clusterMean{cluster: cluster, mean: mean})
	}

	// Sort by mean values in descending order (most decreasing first)
	sort.Slice(clusterMeansList, func(i, j int) bool {
		return clusterMeansList[i].mean > clusterMeansList[j].mean
	})

	// Create the result map with sort order
	result := make(map[int]int)
	for i, cm := range clusterMeansList {
		result[cm.cluster] = i + 1 // Sort order starts from 1
	}

	return result
}
