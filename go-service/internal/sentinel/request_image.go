package sentinel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/airbusgeo/godal"
	pb "github.com/forest-guardian/forest-guardian-api-poc/internal/ml/protobufs"
	"github.com/schollz/progressbar/v3"
	"google.golang.org/grpc"
)

func calculatePixels(distance float64, resolution float64) int {
	return int(distance * (111_000.0 / resolution)) // Rough conversion assuming degrees to meters
}

func batchDates(dates []string, batchSize int) [][]string {
	var batches [][]string
	for batchSize < len(dates) {
		dates, batches = dates[batchSize:], append(batches, dates[0:batchSize:batchSize])
	}
	batches = append(batches, dates)
	return batches
}

func requestImage(startDate, endDate time.Time, geometries []*godal.Geometry) (map[string]map[int]map[time.Time][]byte, error) {
	// Convert geometries to GeoJSON strings
	var geojsonFeatures []string
	for _, g := range geometries {
		geojson, err := g.GeoJSON()
		if err != nil {
			return nil, fmt.Errorf("failed to export geometry to GeoJSON: %w", err)
		}
		geojsonFeatures = append(geojsonFeatures, geojson)
	}

	bands := []string{"B05", "B08", "B11", "B02", "B04", "B06", "CLD", "SCL"}
	grpcAddr := "localhost:50051"

	conn, err := grpc.Dial(grpcAddr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()
	client := pb.NewImageServiceClient(conn)

	// List available dates
	listReq := &pb.ListAvailableDatesRequest{
		GeojsonFeatures: geojsonFeatures,
		Bands:           bands,
		StartDate:       startDate.Format("2006-01-02"),
		EndDate:         endDate.Format("2006-01-02"),
	}
	listResp, err := client.ListAvailableDates(context.Background(), listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list available dates: %v", err)
	}
	if len(listResp.AvailableDates) == 0 {
		return nil, fmt.Errorf("no available dates found for the given range")
	}

	batchSize := 1
	batches := batchDates(listResp.AvailableDates, batchSize)

	var (
		mu     sync.Mutex
		images = make(map[string]map[int]map[time.Time][]byte) // band -> geometry idx -> date -> bytes
		wg     sync.WaitGroup
	)

	progressbar := progressbar.Default(int64(len(batches)*len(bands)), "Requesting images")

	for _, batch := range batches {
		wg.Add(1)
		go func(batch []string) {
			defer wg.Done()
			for _, availableDate := range batch {
				var bandWg sync.WaitGroup
				for _, band := range bands {
					bandWg.Add(1)
					go func(band string) {
						defer bandWg.Done()
						getReq := &pb.GetBandValuesRequest{
							GeojsonFeatures: geojsonFeatures,
							Bands:           []string{band},
							Date:            availableDate,
						}
						getResp, err := client.GetBandValues(context.Background(), getReq)
						if err != nil {
							fmt.Printf("failed to get band values for %s band %s: %v\n", availableDate, band, err)
							return
						}
						parsedDate, err := time.Parse("2006-01-02", availableDate)
						if err != nil {
							fmt.Printf("failed to parse available date: %v\n", err)
							return
						}
						mu.Lock()
						if images[band] == nil {
							images[band] = make(map[int]map[time.Time][]byte)
						}
						for key, bandData := range getResp.BandData {
							// key is like "B04_geom0"
							var idx int
							_, err := fmt.Sscanf(key, band+"_geom%d", &idx)
							if err == nil {
								if images[band][idx] == nil {
									images[band][idx] = make(map[time.Time][]byte)
								}
								images[band][idx][parsedDate] = bandData
							}
						}
						mu.Unlock()
						progressbar.Add(1)
					}(band)
				}
				bandWg.Wait()
			}
		}(batch)
	}
	wg.Wait()
	return images, nil
}
