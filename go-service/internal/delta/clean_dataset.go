package delta

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delta/protobufs"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/sentinel"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/utils"
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
func cleanDataset(pixelDataset map[[2]int]map[time.Time]PixelData) (map[[2]int]map[time.Time]PixelData, error) {
	var (
		mu          sync.Mutex
		cleanData   = make(map[[2]int]map[time.Time]PixelData)
		progressBar = progressbar.Default(int64(len(pixelDataset)), "Cleaning dataset")
	)

	wp := workerpool.New(100)
	conn, err := grpc.Dial(fmt.Sprintf("localhost:%d", properties.GrpcPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	errChan := make(chan error, 1)
	var stopProcessing sync.Once

	for _, data := range pixelDataset {
		d := data // capture range variable
		wp.Submit(func() {
			var ndre, ndmi, psri, ndvi []float64
			ascDates := utils.GetSortedKeys(d, true)
			for _, date := range ascDates {
				pixel := d[date]
				if pixel.Status == sentinel.PixelStatusInvalid {
					panic("invalid pixel found during cleaning")
				}
				if pixel.Status == sentinel.PixelStatusTreatable {
					panic("treatable pixel found during cleaning")
				}
				ndre = append(ndre, pixel.NDRE)
				ndmi = append(ndmi, pixel.NDMI)
				psri = append(psri, pixel.PSRI)
				ndvi = append(ndvi, pixel.NDVI)
			}
			if len(ndre) != len(d) {
				return
			}
			values := map[string][]float64{
				"ndre": ndre, "ndmi": ndmi, "psri": psri, "ndvi": ndvi,
			}

			smoothed, err := clearAndSmooth(conn, values)
			if err != nil {
				stopProcessing.Do(func() { errChan <- err })
				return
			}

			var validData = make(map[time.Time]PixelData)
			for i, date := range ascDates {
				pixel := d[date]
				if smoothed["ndmi"][i] == 0 || smoothed["psri"][i] == 0 || smoothed["ndre"][i] == 0 || smoothed["ndvi"][i] == 0 {
					continue
				}
				pixel.NDMI = smoothed["ndmi"][i]
				pixel.PSRI = smoothed["psri"][i]
				pixel.NDRE = smoothed["ndre"][i]
				pixel.NDVI = smoothed["ndvi"][i]

				validData[date] = pixel
			}

			mu.Lock()
			for date := range validData {
				key := [2]int{d[date].X, d[date].Y}
				if _, ok := cleanData[key]; !ok {
					cleanData[key] = make(map[time.Time]PixelData)
				}
				cleanData[key][date] = validData[date]
			}
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

	if len(cleanData) == 0 {
		return nil, fmt.Errorf("no valid data found after cleaning")
	}
	return cleanData, nil
}
