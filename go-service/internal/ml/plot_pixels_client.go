package ml

import (
	"context"
	"fmt"
	"log"

	pb "github.com/forest-guardian/forest-guardian-api-poc/internal/ml/protobufs"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
	"google.golang.org/grpc"
)

func PlotPixels(forestName, plotID string, ndvi, ndre, ndmi map[string]float64, pixels []*pb.Pixel) {
	conn, err := grpc.Dial(fmt.Sprintf("localhost:%d", properties.GrpcPort), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewPlotPixelsServiceClient(conn)

	req := &pb.PlotPixelsRequest{
		NDVI:       ndvi,
		NDRE:       ndre,
		NDMI:       ndmi,
		Pixels:     pixels,
		ForestName: forestName,
		PlotId:     plotID,
	}

	r, err := c.PlotPixels(context.Background(), req)
	if err != nil {
		log.Fatalf("could not plot pixels: %v", err)
	}

	fmt.Println(r.Message)
}
