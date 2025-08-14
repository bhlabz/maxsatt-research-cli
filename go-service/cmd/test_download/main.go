package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/airbusgeo/godal"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/sentinel"
	"github.com/joho/godotenv"
)

func main() {
	// Hardcoded test parameters - modify these to test different scenarios
	forest := "Atacado-Formiga"
	plot := "1"
	testDate := time.Date(2022, 3, 6, 0, 0, 0, 0, time.UTC)
	intervalDays := 5

	fmt.Println("=== Maxsatt Test Image Download ===")
	fmt.Printf("Forest: %s\n", forest)
	fmt.Printf("Plot: %s\n", plot)
	fmt.Printf("Date: %s\n", testDate.Format("2006-01-02"))
	fmt.Println()

	// Load environment variables
	if err := godotenv.Load("../../.env"); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
		fmt.Println("Make sure you have set the required environment variables:")
		fmt.Println("- COPERNICUS_CLIENT_ID")
		fmt.Println("- COPERNICUS_CLIENT_SECRET")
		fmt.Println("- COPERNICUS_TOKEN_URL")
		fmt.Println("- ROOT_PATH")
		fmt.Println()
	}

	// Set ROOT_PATH if not already set
	if os.Getenv("ROOT_PATH") == "" {
		rootPath := "/Users/gabihert/Documents/Projects/forest-guardian/maxsatt-research-cli"
		os.Setenv("ROOT_PATH", rootPath)
		fmt.Printf("Setting ROOT_PATH to: %s\n", rootPath)
	}

	// Initialize GDAL
	godal.RegisterAll()

	// Get geometry from GeoJSON file
	fmt.Printf("Loading geometry for forest '%s', plot '%s'...\n", forest, plot)
	geometry, err := sentinel.GetGeometryFromGeoJSON(forest, plot)
	if err != nil {
		log.Fatalf("Failed to get geometry: %v", err)
	}
	fmt.Println("✓ Geometry loaded successfully")

	// Calculate date range (single day with interval)
	startDate := testDate
	endDate := testDate

	fmt.Printf("Requesting images from %s to %s (interval: %d days)...\n",
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"),
		intervalDays)

	// Call GetImages function
	images, err := sentinel.GetImages(geometry, forest, plot, startDate, endDate, intervalDays)
	if err != nil {
		log.Fatalf("Failed to get images: %v", err)
	}

	// Display results
	fmt.Printf("\n=== Results ===\n")
	fmt.Printf("Total images downloaded: %d\n", len(images))

	if len(images) == 0 {
		fmt.Println("No images were downloaded. This could mean:")
		fmt.Println("- The image already exists in cache")
		fmt.Println("- No satellite data available for this date")
		fmt.Println("- All pixels were invalid (clouds, etc.)")
		fmt.Println("- API credentials issue")
	} else {
		fmt.Println("\nDownloaded images:")
		for date, dataset := range images {
			fmt.Printf("- %s", date.Format("2006-01-02"))

			// Get image information
			if bounds, err := dataset.Bounds(); err == nil {
				fmt.Printf(" (bounds: %.6f, %.6f, %.6f, %.6f)", bounds[0], bounds[1], bounds[2], bounds[3])
			}

			if size := dataset.Structure(); size.SizeX > 0 && size.SizeY > 0 {
				fmt.Printf(" (size: %dx%d)", size.SizeX, size.SizeY)
			}

			fmt.Printf(" (bands: %d)", dataset.Structure().NBands)
			fmt.Println()
		}
	}

	// Show file location
	imagePath := fmt.Sprintf("%s/data/images/%s_%s", properties.RootPath(), forest, plot)
	fmt.Printf("\nImage files saved to: %s\n", imagePath)

	// Check if any files exist in the directory
	if entries, err := os.ReadDir(imagePath); err == nil {
		fmt.Printf("Files in directory: %d\n", len(entries))
		for _, entry := range entries {
			if !entry.IsDir() {
				fmt.Printf("- %s\n", entry.Name())
			}
		}
	}

	fmt.Println("\n✓ Test completed successfully!")
}
