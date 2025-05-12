package output

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"strings"

	"github.com/airbusgeo/godal"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/delta"
)

func normalize(value, min, max float64) float64 {
	if max == min {
		return 0
	}
	norm := (value - min) / (max - min)
	if norm < 0 {
		return 0
	}
	if norm > 1 {
		return 1
	}
	return norm
}

func valueToColor(norm float64) color.RGBA {
	var r, g, b uint8
	if norm <= 0.5 {
		// Transition from blue to green
		ratio := norm / 0.5
		r = 0
		g = uint8(255 * ratio)
		b = uint8(255 * (1 - ratio))
	} else {
		// Transition from green to red
		ratio := (norm - 0.5) / 0.5
		r = uint8(255 * ratio)
		g = uint8(255 * (1 - ratio))
		b = 0
	}
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

func getPixelIndex(index string, pixel delta.PixelData) float64 {
	switch index {
	case "NDRE":
		return pixel.NDRE
	case "NDMI":
		return pixel.NDMI
	case "PSRI":
		return pixel.PSRI
	case "NDVI":
		return pixel.NDVI
	default:
		return 0
	}
}

func CreateCleanDataImage(result []delta.PixelData, tiffImagePath, outputImagePath string) (string, error) {
	for _, index := range []string{"NDRE", "NDMI", "PSRI", "NDVI"} {
		outputImagePathCpy := outputImagePath + "_" + index
		if !strings.Contains(outputImagePathCpy, ".jpeg") {
			outputImagePathCpy += ".jpeg"
		}
		// Open the TIFF image to get its dimensions
		tiffFile, err := os.Open(tiffImagePath)
		if err != nil {
			fmt.Printf("Error opening TIFF file: %v\n", err)
			return "", err
		}
		defer tiffFile.Close()

		ds, err := godal.Open(tiffImagePath, godal.ErrLogger(func(ec godal.ErrorCategory, code int, msg string) error {
			if ec == godal.CE_Warning {
				return nil
			}
			return err
		}))
		if err != nil {
			fmt.Println(err.Error())

		}

		width, height := int(ds.Structure().SizeX), int(ds.Structure().SizeY)
		// Create a new RGBA image
		newImage := image.NewRGBA(image.Rect(0, 0, width, height))

		// Map the PixelResult to the new image
		for _, pixel := range result {
			if pixel.X >= 0 && pixel.X < width && pixel.Y >= 0 && pixel.Y < height {
				index := getPixelIndex(index, pixel)
				norm := normalize(index, 0, 1)
				clr := valueToColor(norm)
				newImage.Set(pixel.X, pixel.Y, clr)
			}
		}

		outputFile, err := os.Create(outputImagePathCpy)
		if err != nil {
			fmt.Printf("Error creating JPEG file: %v\n", err)
			return "", nil
		}
		defer outputFile.Close()

		err = jpeg.Encode(outputFile, newImage, &jpeg.Options{
			Quality: 100,
		})
		if err != nil {
			fmt.Printf("Error encoding JPEG file: %v\n", err)
			return "", err
		}

		fmt.Println("JPEG image created successfully as", outputImagePathCpy)
	}

	return outputImagePath + "_{index}", nil
}
