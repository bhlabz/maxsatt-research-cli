package output

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"os"
	"time"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/dataset"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
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

func getPixelIndex(index string, pixel dataset.PixelData) float64 {
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

func CreateCleanDataImage(result []dataset.PixelData, forest, plot string, date time.Time) ([]string, error) {

	resultPath := fmt.Sprintf("%s/data/result/%s/%s/clean", properties.RootPath(), forest, plot)

	err := os.MkdirAll(resultPath, os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create result folder: %v", err)
	}

	imagePaths := []string{}
	for _, index := range []string{"NDRE", "NDMI", "PSRI", "NDVI"} {
		resultImagePath := fmt.Sprintf("%s/images/%s", resultPath, index)
		err = os.MkdirAll(resultImagePath, os.ModePerm)
		if err != nil {
			log.Fatalf("Failed to create result folder: %v", err)
		}

		outputPath := fmt.Sprintf("%s/%s_%s_%s.jpeg", resultImagePath, forest, plot, date.Format("2006_01_02"))

		minX, maxX := result[0].X, result[0].X
		minY, maxY := result[0].Y, result[0].Y

		for _, sample := range result {
			if sample.X < minX {
				minX = sample.X
			}
			if sample.X > maxX {
				maxX = sample.X
			}
			if sample.Y < minY {
				minY = sample.Y
			}
			if sample.Y > maxY {
				maxY = sample.Y
			}
		}

		// Calculate image dimensions (add padding)
		width := maxX - minX + 1
		height := maxY - minY + 1
		newImage := image.NewRGBA(image.Rect(0, 0, width, height))

		for _, pixel := range result {
			if pixel.X >= 0 && pixel.X < width && pixel.Y >= 0 && pixel.Y < height {
				if pixel.Color == nil {
					index := getPixelIndex(index, pixel)
					norm := normalize(index, 0, 1)
					clr := valueToColor(norm)
					pixel.Color = &clr
				}

				newImage.Set(pixel.X, pixel.Y, *pixel.Color)
			}
		}

		outputFile, err := os.Create(outputPath)
		if err != nil {
			fmt.Printf("Error creating JPEG file: %v\n", err)
			return nil, nil
		}
		defer outputFile.Close()

		err = jpeg.Encode(outputFile, newImage, &jpeg.Options{
			Quality: 100,
		})
		if err != nil {
			fmt.Printf("Error encoding JPEG file: %v\n", err)
			return nil, err
		}

		fmt.Println("JPEG image created successfully as", outputPath)
		imagePaths = append(imagePaths, outputPath)
	}

	return imagePaths, nil
}
