package ui

import (
	"fmt"
	"os"
)

type menuOption struct {
	title   string
	handler func()
}

// ShowMenu displays the main menu and handles user input
func ShowMenu() {
	menuOptions := []menuOption{
		{"Analyze pest infestation in a forest plot for a specific date", AnalyzePlot},
		{"Analyze pest infestation in forest for a specific date", AnalyzeForest},
		{"Analyze forest plot image indices over time", AnalyzeIndices},
		{"Create a new dataset", CreateDataset},
		{"View the list of available forests", ListForests},
		{"View the list of available forest plots", func() { ListPlots("") }},
		{"Analyze forest plot image deforestation spread over time", AnalyzeSpread},
		{"Plot pixel values over time", PlotPixels},
		{"Exit the application", func() { fmt.Println("Exiting..."); os.Exit(0) }},
	}

	for {
		fmt.Println("\033[34m===================\033[0m")
		for i, opt := range menuOptions {
			fmt.Printf("\033[34m%d. %s\033[0m\n", i+1, opt.title)
		}
		fmt.Println("\033[34mPlease enter your choice:\033[0m")

		var choice int
		_, err := fmt.Scan(&choice)
		if err != nil {
			fmt.Printf("\n\033[31mInvalid input. Please enter a number.\033[0m\n")
			fmt.Scanln() // Clear the buffer
			continue
		}

		if choice < 1 || choice > len(menuOptions) {
			fmt.Println("\033[31mInvalid choice. Please try again.\033[0m")
			continue
		}

		menuOptions[choice-1].handler()
	}
}
