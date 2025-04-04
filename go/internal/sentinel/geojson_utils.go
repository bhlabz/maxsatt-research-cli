package sentinel

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func flattenCoordinates(coordinates interface{}) [][]float64 {
	var flatCoordinates [][]float64

	switch coords := coordinates.(type) {
	case []interface{}:
		for _, coord := range coords {
			switch c := coord.(type) {
			case []interface{}:
				if _, ok := c[0].([]interface{}); ok {
					flatCoordinates = append(flatCoordinates, flattenCoordinates(c)...)
				} else {
					var point []float64
					for _, val := range c {
						point = append(point, val.(float64))
					}
					flatCoordinates = append(flatCoordinates, point)
				}
			}
		}
	}

	return flatCoordinates
}

func GetCentroidLatitudeLongitude(geometry map[string]interface{}) (float64, float64) {
	flatCoordinates := flattenCoordinates(geometry["coordinates"])

	var latitude, longitude float64
	for _, coordinate := range flatCoordinates {
		latitude += coordinate[1]
		longitude += coordinate[0]
	}

	latitude /= float64(len(flatCoordinates))
	longitude /= float64(len(flatCoordinates))

	return latitude, longitude
}

func GetGeometryFromGeoJSON(farm, plot string) (map[string]interface{}, error) {
	filePath := fmt.Sprintf("geojsons/%s.geojson", farm)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	byteValue, _ := io.ReadAll(file)

	var geojson map[string]interface{}
	if err := json.Unmarshal(byteValue, &geojson); err != nil {
		return nil, err
	}

	features := geojson["features"].([]interface{})
	for _, feature := range features {
		featureMap := feature.(map[string]interface{})
		properties := featureMap["properties"].(map[string]interface{})
		if properties["plot_id"] == plot {
			return featureMap["geometry"].(map[string]interface{}), nil
		}
	}

	return nil, fmt.Errorf("geometry not found for farm %s and plot %s", farm, plot)
}

func getAllPlotsAndGeometries(farm string) ([]map[string]interface{}, error) {
	filePath := fmt.Sprintf("geojsons/%s.geojson", farm)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	byteValue, _ := io.ReadAll(file)

	var geojson map[string]interface{}
	if err := json.Unmarshal(byteValue, &geojson); err != nil {
		return nil, err
	}

	var plotsAndGeometries []map[string]interface{}
	features := geojson["features"].([]interface{})
	for _, feature := range features {
		featureMap := feature.(map[string]interface{})
		properties := featureMap["properties"].(map[string]interface{})
		geometry := featureMap["geometry"].(map[string]interface{})
		plotsAndGeometries = append(plotsAndGeometries, map[string]interface{}{
			"plot":     properties["plot_id"],
			"geometry": geometry,
		})
	}

	return plotsAndGeometries, nil
}
