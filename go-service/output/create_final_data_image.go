package output

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/airbusgeo/godal"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/ml"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
)

func CreateFinalDataImage(result []ml.PixelResult, tiffImagePath, outputImageName string) (string, error) {
	outputImagePath := fmt.Sprintf("%s/data/result/%s_final.png", properties.RootPath(), outputImageName)
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
		x, y := int(pixel.X), int(pixel.Y)
		// Find the maximum probability in the result
		maxProbability := 0.0
		label := ""
		for _, pixelResult := range pixel.Result {
			if pixelResult.Probability > maxProbability {
				maxProbability = pixelResult.Probability
				label = pixelResult.Label
			}
		}

		if x >= 0 && x < width && y >= 0 && y < height {
			newImage.Set(int(x), int(y), color.RGBA{
				R: properties.ColorMap[label].R,
				G: properties.ColorMap[label].G,
				B: properties.ColorMap[label].B,
				A: 255,
			})
		}
	}

	// Save the new image as a PNG
	outputFile, err := os.Create(outputImagePath)
	if err != nil {
		fmt.Printf("Error creating PNG file: %v\n", err)
		return "", nil
	}
	defer outputFile.Close()

	err = png.Encode(outputFile, newImage)
	if err != nil {
		fmt.Printf("Error encoding PNG file: %v\n", err)
		return "", err
	}

	fmt.Println("PNG image created successfully as", outputImagePath)
	return outputImagePath, nil
}
