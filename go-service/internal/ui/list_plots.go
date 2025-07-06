package ui

import (
	"fmt"
)

// ListPlots handles the UI for viewing the list of available forest plots
func ListPlots(forest string) {
	PrintWarning("To add a plot to a forest add the 'plot_id' property at the '.geojson' file from the forest fo your choice.\nThe 'plot_id' property should be located at 'features[N]properties.plot_id'.")

	if forest == "" {
		forest = ReadString("Enter the forest name: ")
	}

	plotIDs, err := GetPlotIDsFromGeoJSON(forest)
	if err != nil {
		PrintError(err.Error())
		return
	}

	fmt.Printf("\n%sAvailable plots:%s\n", ColorGreen, ColorReset)
	for _, plotID := range plotIDs {
		fmt.Printf("%s- %s%s\n", ColorGreen, plotID, ColorReset)
	}
}
