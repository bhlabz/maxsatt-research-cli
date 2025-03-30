package sentinel

import (
	"errors"
	"fmt"
	"math"

	"github.com/fogleman/gg"
	"github.com/lukeroth/gdal"
)

func latLonToXY(tiffPath string, lat, lon float64) (int, int, error) {
	dataset, err := gdal.Open(tiffPath, gdal.Access(gdal.ReadOnly))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open TIFF file: %v", err)
	}
	defer dataset.Close()

	geoTransform := dataset.GeoTransform()
	if len(geoTransform) == 0 {
		return 0, 0, errors.New("failed to retrieve GeoTransform")
	}

	// Get dataset bounds
	xMin := geoTransform[0]
	yMax := geoTransform[3]
	xMax := xMin + geoTransform[1]*float64(dataset.RasterXSize())
	yMin := yMax + geoTransform[5]*float64(dataset.RasterYSize())

	// Check if latitude and longitude are within bounds
	if lon < xMin || lon > xMax || lat < yMin || lat > yMax {
		return 0, 0, fmt.Errorf("latitude %f and longitude %f are out of bounds for the image", lat, lon)
	}

	// Convert geographic coordinates (lon, lat) to pixel coordinates
	col := int(math.Floor((lon - geoTransform[0]) / geoTransform[1]))
	row := int(math.Floor((lat - geoTransform[3]) / geoTransform[5]))

	// Validate pixel coordinates within image dimensions
	if col >= 0 && col < dataset.RasterXSize() && row >= 0 && row < dataset.RasterYSize() {
		return col, row, nil
	}
	return 0, 0, fmt.Errorf("pixel coordinates (%d, %d) are out of image bounds", col, row)
}

func xyToLatLon(tiffPath string, x, y int) (float64, float64, error) {
	dataset, err := gdal.Open(tiffPath, gdal.Access(gdal.ReadOnly))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open TIFF file: %v", err)
	}
	defer dataset.Close()

	geoTransform := dataset.GeoTransform()
	if len(geoTransform) == 0 {
		return 0, 0, errors.New("failed to retrieve GeoTransform")
	}

	// Convert pixel coordinates (col, row) to geographic coordinates (lon, lat)
	lon := geoTransform[0] + float64(x)*geoTransform[1]
	lat := geoTransform[3] + float64(y)*geoTransform[5]

	return lon, lat, nil
}

func plotPixelOnImage(tiffPath string, x, y int) error {
	dataset, err := gdal.Open(tiffPath, gdal.Access(gdal.ReadOnly))
	if err != nil {
		return fmt.Errorf("failed to open TIFF file: %v", err)
	}
	defer dataset.Close()

	band := dataset.RasterBand(1)

	width := dataset.RasterXSize()
	height := dataset.RasterYSize()
	data := make([]float64, width*height)
	err = band.IO(gdal.RWFlag(gdal.Read), 0, 0, width, height, data, width, height, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to read raster data: %v", err)
	}

	// Create an image and plot the pixel
	dc := gg.NewContext(width, height)
	for i := 0; i < height; i++ {
		for j := 0; j < width; j++ {
			value := data[i*width+j]
			gray := uint8(value / 256)
			dc.SetRGB(float64(gray)/255, float64(gray)/255, float64(gray)/255)
			dc.SetPixel(j, i)
		}
	}
	dc.SetRGB(1, 0, 0) // Red color for the pixel
	dc.DrawCircle(float64(x), float64(y), 5)
	dc.Fill()

	err = dc.SavePNG("output.png")
	if err != nil {
		return fmt.Errorf("failed to save image: %v", err)
	}

	return nil
}
