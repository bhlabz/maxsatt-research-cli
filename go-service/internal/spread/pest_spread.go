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

func PestSpread(deltaData map[[2]int]map[time.Time]delta.Data) (map[time.Time][]PestSpreadSample, error) {
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

	for date, pixels := range groupedData {
		fmt.Println("Processing pixels for date:", pixels[0].StartDate, "to", pixels[0].EndDate)
		fmt.Printf("Sending %d pixels to clustering service\n", len(pixels))

		// Call the clustering function for each group of pixels
		pestSpreadSamples, err := client.ClusterizeSpread(pixels)
		if err != nil {
			fmt.Printf("Error clustering pixels for date %s: %v\n", date, err)
			continue
		}

		fmt.Printf("Received %d clustered samples for date %s\n", len(pestSpreadSamples), date)

		// Log cluster information
		clusterCounts := make(map[int]int)
		for _, sample := range pestSpreadSamples {
			clusterCounts[sample.Cluster]++
		}

		fmt.Printf("Cluster distribution for date %s:\n", date)
		for cluster, count := range clusterCounts {
			fmt.Printf("  Cluster %d: %d samples\n", cluster, count)
		}

		if allPestSpreadSamples[date] == nil {
			allPestSpreadSamples[date] = []PestSpreadSample{}
		}

		allPestSpreadSamples[date] = append(allPestSpreadSamples[date], pestSpreadSamples...)
	}

	fmt.Printf("Total pest spread samples processed: %d\n", len(allPestSpreadSamples))

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
