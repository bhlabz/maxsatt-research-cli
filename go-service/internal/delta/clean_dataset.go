package delta

import (
	"context"
	"fmt"
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

func clearAndSmooth(conn *grpc.ClientConn, values map[string][]float64) (map[string][]float64, error) {

	client := protobufs.NewClearAndSmoothServiceClient(conn)

	// Create the request
	req := &protobufs.ClearAndSmoothRequest{
		Data: convertToProtobufList(values),
	}

	resp, err := client.ClearAndSmooth(context.Background(), req)
	if err != nil {
		return nil, err
	}

	// Return the smoothed data
	return convertFromProtobufList(resp.SmoothedData), nil
}
func cleanDataset(pixelDataset []PixelData) ([]PixelData, error) {
	groupedData := make(map[[2]int][]PixelData)

	for _, data := range pixelDataset {
		key := [2]int{data.X, data.Y}
		groupedData[key] = append(groupedData[key], data)
	}

	var (
		mu          sync.Mutex
		newArray    []PixelData
		progressBar = progressbar.Default(int64(len(groupedData)), "Cleaning dataset")
	)

	wp := workerpool.New(100)
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	errChan := make(chan error, 1)
	var stopProcessing sync.Once

	for _, data := range groupedData {
		d := data // capture range variable
		wp.Submit(func() {
			var ndre, ndmi, psri, ndvi []float64
			for _, val := range d {
				ndre = append(ndre, val.NDRE)
				ndmi = append(ndmi, val.NDMI)
				psri = append(psri, val.PSRI)
				ndvi = append(ndvi, val.NDVI)
			}

			values := map[string][]float64{
				"ndre": ndre, "ndmi": ndmi, "psri": psri, "ndvi": ndvi,
			}

			smoothed, err := clearAndSmooth(conn, values)
			if err != nil {
				stopProcessing.Do(func() { errChan <- err })
				return
			}

			validData := []PixelData{}
			for i := range d {
				if smoothed["ndmi"][i] == 0 || smoothed["psri"][i] == 0 || smoothed["ndre"][i] == 0 || smoothed["ndvi"][i] == 0 {
					continue
				}
				d[i].NDMI = smoothed["ndmi"][i]
				d[i].PSRI = smoothed["psri"][i]
				d[i].NDRE = smoothed["ndre"][i]
				d[i].NDVI = smoothed["ndvi"][i]
				validData = append(validData, d[i])
			}

			mu.Lock()
			newArray = append(newArray, validData...)
			progressBar.Add(1)
			mu.Unlock()
		})
	}

	// Wait for all tasks
	go func() {
		wp.StopWait()
		close(errChan)
	}()

	// Return the first error if any
	if err := <-errChan; err != nil {
		return nil, fmt.Errorf("error during dataset cleaning: %v", err)
	}

	if len(newArray) == 0 {
		return nil, fmt.Errorf("no valid data found after cleaning")
	}
	return newArray, nil
}
