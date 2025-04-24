package delta

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delta/protobufs"
	"github.com/gammazero/workerpool"
	"github.com/schollz/progressbar/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func convertToProtobufList(data map[string][]float64) map[string]*protobufs.DoubleList {
	result := make(map[string]*protobufs.DoubleList)
	for key, values := range data {
		result[key] = &protobufs.DoubleList{
			Values: values,
		}
	}
	return result
}

func convertFromProtobufList(data map[string]*protobufs.DoubleList) map[string][]float64 {
	result := make(map[string][]float64)
	for key, doubleList := range data {
		result[key] = doubleList.Values
	}
	return result
}

func clearAndSmooth(conn *grpc.ClientConn, values map[string][]float64) map[string][]float64 {

	client := protobufs.NewClearAndSmoothServiceClient(conn)

	// Create the request
	req := &protobufs.ClearAndSmoothRequest{
		Data: convertToProtobufList(values),
	}

	resp, err := client.ClearAndSmooth(context.Background(), req)
	if err != nil {
		log.Fatalf("Failed to call ClearAndSmooth: %v", err)
	}

	// Return the smoothed data
	return convertFromProtobufList(resp.SmoothedData)
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
	wp := workerpool.New(100)
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()
	for _, data := range groupedData {
		wg.Add(1)

		wp.Submit(func() {
			defer wg.Done()
			var ndre, ndmi, psri, ndvi []float64
			for _, d := range data {
				ndre = append(ndre, d.NDRE)
				ndmi = append(ndmi, d.NDMI)
				psri = append(psri, d.PSRI)
				ndvi = append(ndvi, d.NDVI)
			}

			values := map[string][]float64{
				"ndre": ndre,
				"ndmi": ndmi,
				"psri": psri,
				"ndvi": ndvi,
			}

			values = clearAndSmooth(conn, values)

			ndre = values["ndre"]
			ndmi = values["ndmi"]
			psri = values["psri"]
			ndvi = values["ndvi"]

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
		})

	}

	wg.Wait()

	if len(newArray) > 0 {
		return newArray
	} else {
		fmt.Println("No valid data found")
		return nil
	}
}
