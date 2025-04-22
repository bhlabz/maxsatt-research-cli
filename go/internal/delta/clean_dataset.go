package delta

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delta/protobufs"
	"github.com/schollz/progressbar/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func clearAndSmooth(values []float64) []float64 {

	// Connect to the gRPC server
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	client := protobufs.NewClearAndSmoothServiceClient(conn)

	// Create the request
	req := &protobufs.ClearAndSmoothRequest{
		Data: values,
	}

	// Call the gRPC method
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.ClearAndSmooth(ctx, req)
	if err != nil {
		log.Fatalf("Failed to call ClearAndSmooth: %v", err)
	}

	// Return the smoothed data
	return resp.SmoothedData
}
func cleanDataset(pixelDataset []PixelData) []PixelData {
	groupedData := make(map[[2]int][]PixelData)

	// Group data by (x, y)
	for _, data := range pixelDataset {
		key := [2]int{data.X, data.Y}
		groupedData[key] = append(groupedData[key], data)
	}

	var wg sync.WaitGroup
	mu := sync.Mutex{}
	newArray := []PixelData{}
	progressBar := progressbar.Default(int64(len(groupedData)), "Cleaning dataset")
	// Create a buffered channel to limit the number of goroutines
	goroutineLimit := make(chan struct{}, 100) // Limit to 10 goroutines

	for key, data := range groupedData {
		wg.Add(1)
		goroutineLimit <- struct{}{} // Acquire a slot

		go func(key [2]int, data []PixelData) {
			defer wg.Done()
			defer func() { <-goroutineLimit }() // Release the slot

			var ndre, ndmi, psri, ndvi []float64
			for _, d := range data {
				ndre = append(ndre, d.NDRE)
				ndmi = append(ndmi, d.NDMI)
				psri = append(psri, d.PSRI)
				ndvi = append(ndvi, d.NDVI)
			}

			ndmi = clearAndSmooth(ndmi)
			psri = clearAndSmooth(psri)
			ndre = clearAndSmooth(ndre)
			ndvi = clearAndSmooth(ndvi)

			validData := []PixelData{}
			for i := range data {
				if ndmi[i] == 0 || psri[i] == 0 || ndre[i] == 0 || ndvi[i] == 0 {
					continue
				}
				data[i].NDMI = ndmi[i]
				data[i].PSRI = psri[i]
				data[i].NDRE = ndre[i]
				data[i].NDVI = ndvi[i]
				validData = append(validData, data[i])
			}

			mu.Lock()
			progressBar.Add(1)
			newArray = append(newArray, validData...)
			mu.Unlock()
		}(key, data)
	}

	wg.Wait()

	if len(newArray) > 0 {
		return newArray
	} else {
		fmt.Println("No valid data found")
		return nil
	}
}
