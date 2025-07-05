package output

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"os"
	"sort"
	"time"

	"github.com/fogleman/gg"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/spread"
)

// CreatePestSpreadImage generates an image from PestSpreadSample array
// Each cluster gets a distinct color, and pixels are placed at their X,Y coordinates
func CreatePestSpreadImage(samples []spread.PestSpreadSample, forest, plot string, date time.Time) error {
	resultPath := fmt.Sprintf("%s/data/result/%s/%s/spread", properties.RootPath(), forest, plot)

	err := os.MkdirAll(resultPath, os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create result folder: %v", err)
	}

	resultImagePath := fmt.Sprintf("%s/images", resultPath)
	err = os.MkdirAll(resultImagePath, os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create result folder: %v", err)
	}
	outputPath := fmt.Sprintf("%s/%s_%s_%s.jpg", resultImagePath, forest, plot, date.Format("2006_01_02"))
	if len(samples) == 0 {
		return fmt.Errorf("no samples provided")
	}

	// Find the dimensions of the image
	minX, maxX := samples[0].X, samples[0].X
	minY, maxY := samples[0].Y, samples[0].Y

	for _, sample := range samples {
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

	// Create a color palette for clusters
	clusterColors := generateClusterColors(samples)

	// Create the image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill background with white
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Draw each sample as a pixel with its cluster color
	for _, sample := range samples {
		x := sample.X - minX
		y := sample.Y - minY

		if x >= 0 && x < width && y >= 0 && y < height {
			clusterColor := clusterColors[sample.Cluster]
			img.Set(x, y, clusterColor)
		}
	}

	// Save the image as JPEG
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	err = jpeg.Encode(file, img, &jpeg.Options{Quality: 100})
	if err != nil {
		return fmt.Errorf("failed to encode image: %w", err)
	}

	fmt.Printf("Pest spread image saved to: %s\n", outputPath)
	fmt.Printf("Image dimensions: %dx%d\n", width, height)
	fmt.Printf("Number of clusters: %d\n", len(clusterColors))

	// Print cluster information
	fmt.Println("Cluster colors:")
	for cluster, color := range clusterColors {
		fmt.Printf("  Cluster %d: RGB(%d, %d, %d)\n", cluster, color.R, color.G, color.B)
	}

	return nil
}

// generateClusterColors creates a color palette for each unique cluster based on severity
func generateClusterColors(samples []spread.PestSpreadSample) map[int]color.RGBA {
	// Get unique clusters and their severities
	clusterSeverities := make(map[int]int)
	for _, sample := range samples {
		clusterSeverities[sample.Cluster] = sample.Severity
	}

	// Find the maximum severity to normalize the color scale
	maxSeverity := 1
	for _, severity := range clusterSeverities {
		if severity > maxSeverity {
			maxSeverity = severity
		}
	}

	// Generate colors based on severity
	clusterColors := make(map[int]color.RGBA)
	for cluster, severity := range clusterSeverities {
		// Normalize severity to 0-1 range (1 = red, maxSeverity = green)
		normalizedSeverity := float64(severity-1) / float64(maxSeverity-1)

		// Create gradient from red (severity 1) to green (max severity)
		red := uint8(255 * (1 - normalizedSeverity)) // Red decreases as severity increases
		green := uint8(255 * normalizedSeverity)     // Green increases as severity increases
		blue := uint8(0)                             // No blue component

		clusterColors[cluster] = color.RGBA{
			R: red,
			G: green,
			B: blue,
			A: 255,
		}
	}

	return clusterColors
}

// CreatePestSpreadImageWithLegend creates an image with a legend showing cluster colors
func CreatePestSpreadImageWithLegend(samples []spread.PestSpreadSample, outputPath string) error {
	if len(samples) == 0 {
		return fmt.Errorf("no samples provided")
	}

	// Find the dimensions of the image
	minX, maxX := samples[0].X, samples[0].X
	minY, maxY := samples[0].Y, samples[0].Y

	for _, sample := range samples {
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

	// Calculate image dimensions
	width := maxX - minX + 1
	height := maxY - minY + 1

	// Add space for legend
	legendHeight := 100
	totalHeight := height + legendHeight

	// Create the image with legend space
	img := image.NewRGBA(image.Rect(0, 0, width, totalHeight))

	// Fill background with white
	for y := 0; y < totalHeight; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Generate cluster colors
	clusterColors := generateClusterColors(samples)

	// Draw each sample as a pixel with its cluster color
	for _, sample := range samples {
		x := sample.X - minX
		y := sample.Y - minY

		if x >= 0 && x < width && y >= 0 && y < height {
			clusterColor := clusterColors[sample.Cluster]
			img.Set(x, y, clusterColor)
		}
	}

	// Create a context for drawing the legend
	dc := gg.NewContext(width, totalHeight)
	dc.SetRGB(1, 1, 1) // White background
	dc.Clear()

	// Draw the main image
	dc.DrawImage(img, 0, 0)

	// Draw legend
	legendY := height + 10
	legendX := 10
	legendSpacing := 20

	// Get sorted cluster IDs by severity (lowest severity first)
	var clusterIDs []int
	for cluster := range clusterColors {
		clusterIDs = append(clusterIDs, cluster)
	}
	sort.Slice(clusterIDs, func(i, j int) bool {
		// Find severity for each cluster
		var severityI, severityJ int
		for _, sample := range samples {
			if sample.Cluster == clusterIDs[i] {
				severityI = sample.Severity
				break
			}
		}
		for _, sample := range samples {
			if sample.Cluster == clusterIDs[j] {
				severityJ = sample.Severity
				break
			}
		}
		return severityI < severityJ
	})

	// Draw legend items
	for i, clusterID := range clusterIDs {
		y := legendY + i*legendSpacing

		// Find severity for this cluster
		var severity int
		for _, sample := range samples {
			if sample.Cluster == clusterID {
				severity = sample.Severity
				break
			}
		}

		// Draw color box
		color := clusterColors[clusterID]
		dc.SetRGB(float64(color.R)/255, float64(color.G)/255, float64(color.B)/255)
		dc.DrawRectangle(float64(legendX), float64(y), 15, 15)
		dc.Fill()

		// Draw border
		dc.SetRGB(0, 0, 0)
		dc.DrawRectangle(float64(legendX), float64(y), 15, 15)
		dc.SetLineWidth(1)
		dc.Stroke()

		// Draw text with severity information
		dc.SetRGB(0, 0, 0)
		dc.DrawStringAnchored(fmt.Sprintf("Cluster %d (Severity %d)", clusterID, severity), float64(legendX+20), float64(y+7), 0, 0.5)
	}

	// Save the image as JPEG
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	err = jpeg.Encode(file, dc.Image(), &jpeg.Options{Quality: 90})
	if err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}

	fmt.Printf("Pest spread image with legend saved to: %s\n", outputPath)
	fmt.Printf("Image dimensions: %dx%d\n", width, totalHeight)
	fmt.Printf("Number of clusters: %d\n", len(clusterColors))

	return nil
}
