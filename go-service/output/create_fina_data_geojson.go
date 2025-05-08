package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/ml"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
)

func CreateFinalDataGeoJson(result []ml.PixelResult, outputGeojsonPath string) string {
	outputPath := fmt.Sprintf("%s/data/result/%s.geojson", properties.RootPath(), outputGeojsonPath)
	features := make([]map[string]interface{}, 0)

	for _, pixel := range result {
		results := []interface{}{}
		for _, pixelResult := range pixel.Result {
			results = append(results, map[string]interface{}{
				"label":       pixelResult.Label,
				"probability": pixelResult.Probability,
			})
		}

		feature := map[string]interface{}{
			"type": "Feature",
			"geometry": map[string]interface{}{
				"type":        "Point",
				"coordinates": []float64{pixel.Longitude, pixel.Latitude},
			},
			"properties": map[string]interface{}{
				"results": results,
			},
		}
		features = append(features, feature)
	}

	geoJSON := map[string]interface{}{
		"type":     "FeatureCollection",
		"features": features,
	}

	file, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("Error creating GeoJSON file: %v\n", err)
		return ""
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(geoJSON); err != nil {
		fmt.Printf("Error encoding GeoJSON: %v\n", err)
		return ""
	}

	fmt.Println("GeoJSON file created successfully at", outputPath)
	return outputPath
}
