package delivery

import (
	"fmt"
	"time"

	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/airbusgeo/godal"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/delta"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/final"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/ml"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/sentinel"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/weather"
)

func createImage(result []ml.PixelResult, tiffImagePath, outputImagePath string) {
	outputImagePath = fmt.Sprintf("%s/data/result/%s.png", properties.RootPath(), outputImagePath)
	// Open the TIFF image to get its dimensions
	tiffFile, err := os.Open(tiffImagePath)
	if err != nil {
		fmt.Printf("Error opening TIFF file: %v\n", err)
		return
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
		return
	}
	defer outputFile.Close()

	err = png.Encode(outputFile, newImage)
	if err != nil {
		fmt.Printf("Error encoding PNG file: %v\n", err)
		return
	}

	fmt.Println("PNG image created successfully as", outputImagePath)
}

func EvaluatePlot(farm, plot string, endDate time.Time) (string, error) {
	start := time.Now()

	deltaDays := 5
	deltaDaysTrashHold := 40
	getDaysBeforeEvidenceToAnalyse := deltaDays + deltaDaysTrashHold
	startDate := endDate.AddDate(0, 0, -getDaysBeforeEvidenceToAnalyse)
	outputFileName := fmt.Sprintf("%s_%s_%s", farm, plot, endDate.Format("2006-01-02"))

	stepStart := time.Now()
	geometry, err := sentinel.GetGeometryFromGeoJSON(farm, plot)
	if err != nil {
		return "", err
	}
	fmt.Printf("GetGeometryFromGeoJSON took %v\n", time.Since(stepStart))

	stepStart = time.Now()
	images, err := sentinel.GetImages(geometry, farm, plot, startDate, endDate, 1)
	if err != nil {
		return "", err
	}
	fmt.Printf("GetImages took %v\n", time.Since(stepStart))

	stepStart = time.Now()
	deltaDataset, err := delta.CreateDeltaDataset(farm, plot, images, deltaDays, deltaDaysTrashHold)
	if err != nil {
		return "", err
	}
	fmt.Printf("CreateDeltaDataset took %v\n", time.Since(stepStart))

	latitude, longitude, err := sentinel.GetCentroidLatitudeLongitudeFromGeometry(geometry)
	if err != nil {
		return "", err
	}

	stepStart = time.Now()
	historicalWeather, err := weather.FetchWeather(latitude, longitude, startDate.AddDate(0, -4, 0), endDate, 10)
	if err != nil {
		return "", err
	}
	fmt.Printf("FetchWeather took %v\n", time.Since(stepStart))

	stepStart = time.Now()
	plotFinalDataset, err := final.GetFinalData(deltaDataset, historicalWeather, startDate, endDate, farm, plot, outputFileName)
	if err != nil {
		return "", err
	}
	fmt.Printf("GetFinalData took %v\n", time.Since(stepStart))

	stepStart = time.Now()
	result, err := ml.RunModel(plotFinalDataset)
	if err != nil {
		return "", err
	}
	fmt.Printf("runModel took %v\n", time.Since(stepStart))

	imageFolderPath := fmt.Sprintf("%s/data/images/%s_%s/", properties.RootPath(), farm, plot)

	files, err := os.ReadDir(imageFolderPath)
	if err != nil {
		return "", fmt.Errorf("error reading image folder: %v", err)
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no files found in image folder: %s", imageFolderPath)
	}

	firstFileName := files[0].Name()
	firstFilePath := fmt.Sprintf("%s%s", imageFolderPath, firstFileName)
	fmt.Printf("First file path in the folder: %s\n", firstFilePath)

	createImage(result, firstFilePath, outputFileName)

	fmt.Printf("Total evaluatePlot execution time: %v\n", time.Since(start))
	return outputFileName, nil
}
