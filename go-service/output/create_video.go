package output

import (
	"bytes"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"github.com/icza/mjpeg"
)

func CreateVideoFromImages(imagePaths []string, outputPath string) error {
	if !strings.Contains(outputPath, ".avi") {
		outputPath += ".avi"
	}
	// Open the first image to get dimensions
	firstFile, err := os.Open(imagePaths[0])
	if err != nil {
		return err
	}
	img, _, err := image.Decode(firstFile)
	firstFile.Close()
	if err != nil {
		return err
	}
	bounds := img.Bounds()
	width := int32(bounds.Dx())
	height := int32(bounds.Dy())

	// Create video writer
	writer, err := mjpeg.New(outputPath, width, height, 2)
	if err != nil {
		return err
	}
	defer writer.Close()

	// Encode and add each image
	for _, path := range imagePaths {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			return err
		}

		var buf bytes.Buffer
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 100})
		if err != nil {
			return err
		}

		err = writer.AddFrame(buf.Bytes())
		if err != nil {
			return err
		}

	}

	return nil
}
