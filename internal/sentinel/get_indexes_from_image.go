package sentinel

import (
	"math"

	"github.com/airbusgeo/godal"
)

func GetIndexesFromImage(dataset *godal.Dataset) (map[string][][]float64, error) {
	// Read bands
	bandsData := dataset.Bands()

	// Map specific band names to their corresponding Band objects
	bands := map[string]godal.Band{
		"B05": bandsData[0], // Band 1
		"B08": bandsData[1], // Band 2
		"B11": bandsData[2], // Band 3
		"B02": bandsData[3], // Band 4
		"B04": bandsData[4], // Band 5
		"B06": bandsData[5], // Band 6
		"CLD": bandsData[6], // Band 7
		"SCL": bandsData[7], // Band 8
	}
	// Read data from bands
	readBand := func(band godal.Band) ([][]float64, error) {
		xSize := band.Structure().SizeX
		ySize := band.Structure().SizeY
		data := make([]float64, xSize*ySize)
		err := band.Read(0, 0, data, xSize, ySize)
		if err != nil {
			return nil, err
		}
		result := make([][]float64, ySize)
		for i := range result {
			result[i] = data[i*xSize : (i+1)*xSize]
		}
		return result, nil
	}

	bandData := make(map[string][][]float64)
	for key, band := range bands {
		var err error
		bandData[key], err = readBand(band)
		if err != nil {
			return nil, err
		}
	}

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

	return indexes, nil
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
