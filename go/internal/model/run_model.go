package model

import (
	"context"
	"log"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/final"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/model/protobufs"

	"google.golang.org/grpc"
)

func RunModel(finalData []final.FinalData) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	client := protobufs.NewRunModelServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	req := &protobufs.RunModelRequest{
		Data: convertToProtoFinalData(finalData),
	}

	resp, err := client.RunModel(ctx, req)
	if err != nil {
		log.Fatalf("Error calling RunModel: %v", err)
	}

	log.Printf("RunModel response: %v", resp)
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
