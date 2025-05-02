package ml

import (
	"context"
	"fmt"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/final"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/ml/protobufs"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type LabelProbability struct {
	Label       string
	Probability float64
}
type PixelResult struct {
	X         int32
	Y         int32
	Latitude  float64
	Longitude float64
	Result    []*LabelProbability
}

func RunModel(finalData []final.FinalData) ([]PixelResult, error) {
	conn, err := grpc.NewClient("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(10*1024*1024),
			grpc.MaxCallSendMsgSize(10*1024*1024),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	client := protobufs.NewRunModelServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60*15)
	defer cancel()

	req := &protobufs.RunModelRequest{
		Data: convertToProtoFinalData(finalData),
	}

	resp, err := client.RunModel(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error calling RunModel: %v", err)
	}

	return convertToPixelResult(resp.Results), nil
}

func convertToPixelResult(data []*protobufs.PixelResult) []PixelResult {
	var pixelResults []PixelResult
	for _, pixel := range data {
		var labelProbabilities []*LabelProbability
		for _, result := range pixel.Result {
			labelProbabilities = append(labelProbabilities, &LabelProbability{
				Label:       result.Label,
				Probability: result.Probability,
			})
		}
		pixelResults = append(pixelResults, PixelResult{
			X:         pixel.X,
			Y:         pixel.Y,
			Latitude:  pixel.Latitude,
			Longitude: pixel.Longitude,
			Result:    labelProbabilities,
		})
	}
	return pixelResults
}

func convertToProtoFinalData(data []final.FinalData) []*protobufs.FinalData {
	var protoData []*protobufs.FinalData
	for _, d := range data {
		label := ""
		if d.Label != nil {
			label = *d.Label
		}
		protoData = append(protoData, &protobufs.FinalData{
			Weather: &protobufs.FinalData_WeatherMetrics{
				AvgTemperature:     d.AvgTemperature,
				TempStdDev:         d.TempStdDev,
				AvgHumidity:        d.AvgHumidity,
				HumidityStdDev:     d.HumidityStdDev,
				TotalPrecipitation: d.TotalPrecipitation,
				DryDaysConsecutive: int32(d.DryDaysConsecutive),
			},
			Delta: &protobufs.FinalData_DeltaData{
				Farm:           d.Farm,
				Plot:           d.Plot,
				DeltaMin:       int32(d.DeltaMin),
				DeltaMax:       int32(d.DeltaMax),
				NdreDerivative: d.NDREDerivative,
				NdmiDerivative: d.NDMIDerivative,
				PsriDerivative: d.PSRIDerivative,
				NdviDerivative: d.NDVIDerivative,
				Ndre:           d.NDRE,
				Ndmi:           d.NDMI,
				Psri:           d.PSRI,
				Ndvi:           d.NDVI,
				Delta:          int32(d.Delta),
				X:              int32(d.X),
				Y:              int32(d.Y),
				EndDate:        d.EndDate.Format(time.RFC3339),
				Label:          label,
				StartDate:      d.StartDate.Format(time.RFC3339),
			},
		})
	}
	return protoData
}
