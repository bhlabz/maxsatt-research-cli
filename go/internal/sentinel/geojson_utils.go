package sentinel

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/airbusgeo/godal"
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

func GetCentroidLatitudeLongitudeFromGeometry(g *godal.Geometry) (float64, float64, error) {
	geojsonStr, err := g.GeoJSON()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to export geometry to GeoJSON: %w", err)
	}

	var parsed struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
	}
	if err := json.Unmarshal([]byte(geojsonStr), &parsed); err != nil {
		return 0, 0, fmt.Errorf("invalid geojson: %w", err)
	}

	// Now flatten all coordinates recursively
	var flatCoords [][]float64
	err = json.Unmarshal(parsed.Coordinates, &flatCoords)
	if err != nil {
		// maybe it's multipolygon?
		var multi [][][][]float64
		if err := json.Unmarshal(parsed.Coordinates, &multi); err == nil {
			for _, poly := range multi {
				for _, ring := range poly {
					flatCoords = append(flatCoords, ring...)
				}
			}
		} else {
			return 0, 0, fmt.Errorf("unsupported geometry format")
		}
	}

	var sumX, sumY float64
	for _, pt := range flatCoords {
		if len(pt) != 2 {
			continue
		}
		sumX += pt[0]
		sumY += pt[1]
	}
	n := float64(len(flatCoords))
	if n == 0 {
		return 0, 0, fmt.Errorf("no coordinates found")
	}
	return sumY / n, sumX / n, nil
}
func GetGeometryFromGeoJSON(farm, plot string) (*godal.Geometry, error) {
	filePath := fmt.Sprintf("../data/geojsons/%s.geojson", farm)

	godal.RegisterInternalDrivers()
	ds, err := godal.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer ds.Close()

	layer := ds.Layers()[0]
	for {
		feat := layer.NextFeature()
		if feat == nil {
			break
		}
		defer feat.Close()

		val, ok := feat.Fields()["plot_id"]
		if !ok {
			continue
		}

		if val.String() == plot {
			geom := feat.Geometry()
			wkb, _ := geom.WKB()
			return godal.NewGeometryFromWKB(wkb, geom.SpatialRef())
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
