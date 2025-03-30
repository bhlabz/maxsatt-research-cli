package sentinel

import (
	"math"

	"github.com/lukeroth/gdal"
)

func GetIndexesFromImage(dataset gdal.Dataset) map[string][][]float64 {
	// Read bands
	bands := map[string]gdal.RasterBand{
		"B05": dataset.RasterBand(1),
		"B08": dataset.RasterBand(2),
		"B11": dataset.RasterBand(3),
		"B02": dataset.RasterBand(4),
		"B04": dataset.RasterBand(5),
		"B06": dataset.RasterBand(6),
		"CLD": dataset.RasterBand(7),
		"SCL": dataset.RasterBand(8),
	}

	// Read data from bands
	readBand := func(band gdal.RasterBand) [][]float64 {
		xSize := band.XSize()
		ySize := band.YSize()
		data := make([]float64, xSize*ySize)
		band.IO(gdal.RWFlag(gdal.Read), 0, 0, xSize, ySize, data, xSize, ySize, 0, 0)
		result := make([][]float64, ySize)
		for i := range result {
			result[i] = data[i*xSize : (i+1)*xSize]
		}
		return result
	}

	bandData := make(map[string][][]float64)
	for key, band := range bands {
		bandData[key] = readBand(band)
	}

	// Calculate indexes
	ndre := calculateIndex(bandData["B08"], bandData["B05"])
	ndmi := calculateIndex(bandData["B08"], bandData["B11"])
	psri := calculateIndex(bandData["B04"], bandData["B06"])
	ndvi := calculateIndex(bandData["B08"], bandData["B04"])

	indexes := map[string][][]float64{
		"ndre":  ndre,
		"ndmi":  ndmi,
		"psri":  psri,
		"ndvi":  ndvi,
		"b02":   bandData["B02"],
		"b04":   bandData["B04"],
		"cloud": bandData["CLD"],
		"scl":   bandData["SCL"],
	}

	return indexes
}

func calculateIndex(band1, band2 [][]float64) [][]float64 {
	rows := len(band1)
	cols := len(band1[0])
	result := make([][]float64, rows)
	for i := range result {
		result[i] = make([]float64, cols)
		for j := range result[i] {
			denominator := band1[i][j] + band2[i][j]
			if denominator != 0 {
				result[i][j] = (band1[i][j] - band2[i][j]) / denominator
			} else {
				result[i][j] = 0
			}
		}
	}
	return result
}

func validateIndexes(psri, ndvi, ndmi, ndre, cld, scl, b02, b04 float64, weather map[string]interface{}) bool {
	if math.IsNaN(psri) || math.IsNaN(ndvi) || math.IsNaN(ndmi) || math.IsNaN(ndre) {
		return false
	}
	if cld > 0 || scl == 3 || scl == 8 || scl == 9 || scl == 10 {
		return false
	}
	if (b04+b02)/2 > 0.9 {
		return false
	}
	if weather["precipitation"] == nil || weather["temperature"] == nil {
		return false
	}
	if psri == 0 && ndvi == 0 && ndmi == 0 && ndre == 0 {
		return false
	}
	return true
}
