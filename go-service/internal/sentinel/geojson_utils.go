package sentinel

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/airbusgeo/godal"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
)

func GetCentroidLatitudeLongitudeFromGeometry(g *godal.Geometry) (float64, float64, error) {
	json, err := g.GeoJSON()
	if err != nil {
		return 0, 0, err
	}
	geomT, err := geojson.UnmarshalGeometry([]byte(json))
	if err != nil {
		log.Fatalf("Failed to unmarshal WKB: %v", err)
	}

	// Assert the type to *geom.MultiPolygon
	centroid, area := planar.CentroidArea(geomT.Coordinates)
	if area <= 0 {
		return 0, 0, errors.New("error getting centroid")
	}
	return centroid.Y(), centroid.X(), nil
}

func GetGeometryFromGeoJSON(farm, plot string) (*godal.Geometry, error) {

	filePath := fmt.Sprintf("%s/data/geojsons/%s.geojson", properties.RootPath(), farm)

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
