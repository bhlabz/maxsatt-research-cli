package spread

import (
	"context"
	"fmt"
	"math"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/delta"
	pb "github.com/forest-guardian/forest-guardian-api-poc/internal/spread/protobufs"
)

// PestClusteringClient represents a client for the pest clustering gRPC service
type PestClusteringClient struct {
	client pb.PestClusteringServiceClient
	conn   *grpc.ClientConn
}

// NewPestClusteringClient creates a new client connection to the pest clustering service
func NewPestClusteringClient(serverAddr string) (*PestClusteringClient, error) {
	conn, err := grpc.NewClient(serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to pest clustering service: %w", err)
	}

	client := pb.NewPestClusteringServiceClient(conn)
	return &PestClusteringClient{
		client: client,
		conn:   conn,
	}, nil
}

// Close closes the gRPC connection
func (c *PestClusteringClient) Close() error {
	return c.conn.Close()
}

// ClusterizeSpread sends delta data to the Python server for clustering
func (c *PestClusteringClient) ClusterizeSpread(deltaData []delta.Data) ([]PestSpreadSample, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Convert delta.Data to protobuf DeltaData
	var pbDeltaData []*pb.DeltaData
	for _, data := range deltaData {
		pbData := &pb.DeltaData{
			Farm:           data.Farm,
			Plot:           data.Plot,
			DeltaMin:       int32(data.DeltaMin),
			DeltaMax:       int32(data.DeltaMax),
			Delta:          int32(data.Delta),
			StartDate:      data.StartDate.Format(time.RFC3339),
			EndDate:        data.EndDate.Format(time.RFC3339),
			X:              int32(data.X),
			Y:              int32(data.Y),
			Ndre:           data.NDRE,
			Ndmi:           data.NDMI,
			Psri:           data.PSRI,
			Ndvi:           data.NDVI,
			NdreDerivative: data.NDREDerivative,
			NdmiDerivative: data.NDMIDerivative,
			PsriDerivative: data.PSRIDerivative,
			NdviDerivative: data.NDVIDerivative,
			Latitude:       data.Latitude,
			Longitude:      data.Longitude,
		}
		if data.Label != nil {
			pbData.Label = *data.Label
		}
		pbDeltaData = append(pbDeltaData, pbData)
	}

	// Create the request
	request := &pb.ClusterizeSpreadRequest{
		DeltaData: pbDeltaData,
	}

	// Call the gRPC service
	response, err := c.client.ClusterizeSpread(ctx, request, grpc.MaxCallRecvMsgSize(math.MaxInt32), grpc.MaxCallSendMsgSize(math.MaxInt32))
	if err != nil {
		return nil, fmt.Errorf("failed to call ClusterizeSpread: %w", err)
	}

	// Convert response back to PestSpreadSample
	var pestSpreadSamples []PestSpreadSample
	for _, pbSample := range response.PestSpreadSamples {
		// Parse dates
		startDate, err := time.Parse(time.RFC3339, pbSample.Data.StartDate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse start date: %w", err)
		}
		endDate, err := time.Parse(time.RFC3339, pbSample.Data.EndDate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse end date: %w", err)
		}

		// Convert protobuf DeltaData back to delta.Data
		deltaData := delta.Data{
			Farm:      pbSample.Data.Farm,
			Plot:      pbSample.Data.Plot,
			DeltaMin:  int(pbSample.Data.DeltaMin),
			DeltaMax:  int(pbSample.Data.DeltaMax),
			Delta:     int(pbSample.Data.Delta),
			StartDate: startDate,
			EndDate:   endDate,
			PixelData: delta.PixelData{
				X:         int(pbSample.Data.X),
				Y:         int(pbSample.Data.Y),
				NDRE:      pbSample.Data.Ndre,
				NDMI:      pbSample.Data.Ndmi,
				PSRI:      pbSample.Data.Psri,
				NDVI:      pbSample.Data.Ndvi,
				Latitude:  pbSample.Data.Latitude,
				Longitude: pbSample.Data.Longitude,
			},

			NDREDerivative: pbSample.Data.NdreDerivative,
			NDMIDerivative: pbSample.Data.NdmiDerivative,
			PSRIDerivative: pbSample.Data.PsriDerivative,
			NDVIDerivative: pbSample.Data.NdviDerivative,
		}
		if pbSample.Data.Label != "" {
			deltaData.Label = &pbSample.Data.Label
		}

		sample := PestSpreadSample{
			Data:    deltaData,
			Cluster: int(pbSample.Cluster),
		}
		pestSpreadSamples = append(pestSpreadSamples, sample)
	}

	return pestSpreadSamples, nil
}
