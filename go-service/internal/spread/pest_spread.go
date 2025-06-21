package spread

import (
	"fmt"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delta"
)

type PestSpreadSample struct {
	delta.Data
	Cluster int
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

	return allPestSpreadSamples, nil
}
