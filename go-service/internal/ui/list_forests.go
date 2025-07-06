package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
)

// ListForests handles the UI for viewing the list of available forests
func ListForests() {
	files, err := os.ReadDir(properties.RootPath() + "/data/geojsons")
	if err != nil {
		PrintError(fmt.Sprintf("Error reading geojsons folder: %s", err.Error()))
		return
	}

	PrintWarning("To add a new forest, add its '.geojson' file at 'data/geojsons' folder.")

	fmt.Printf("\n%sAvailable forests:%s\n", ColorGreen, ColorReset)
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".geojson") {
			fmt.Printf("%s- %s%s\n", ColorGreen, strings.TrimSuffix(file.Name(), ".geojson"), ColorReset)
		}
	}
}
