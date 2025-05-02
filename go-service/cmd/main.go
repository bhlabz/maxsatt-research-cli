package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	_ "image/png" // support PNG decoding

	"github.com/common-nighthawk/go-figure"
	"github.com/fatih/color"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/delivery"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/notification"
	"github.com/joho/godotenv"
)

func printBanner() {
	// Print the banner with go-figure
	figure1 := figure.NewFigure("Maxsatt", "isometric1", true)
	figure2 := figure.NewFigure("CLI", "isometric1", true)
	color.Cyan(figure1.String())
	color.Cyan(figure2.String())
	fmt.Println()
}

func initCLI() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\n\033[31mAn error occurred: %v\033[0m\n", r)
			fmt.Printf("\n\033[31mPlease check the input and try again.\033[0m\n")
			fmt.Printf("\n\033[31mIf the problem persists, please contact support.\033[0m\n")
			fmt.Printf("\n\033[31mExiting...\033[0m\n")
			errMessage := fmt.Sprintf("Maxsatt CLI panic:\n\n %v", r)
			err := notification.SendDiscordErrorNotification(errMessage)
			if err != nil {
				fmt.Printf("\n\033[31mFailed to send notification: %s\033[0m\n", err.Error())
			}
		}
	}()
	printBanner()

	for {
		fmt.Println("\033[34m===================\033[0m")
		fmt.Println("\033[34m1. Evaluate a forest plot\033[0m")
		fmt.Println("\033[34m2. Create a new dataset\033[0m")
		fmt.Println("\033[34m3. Exit\033[0m")
		fmt.Println("\033[34mEnter your choice:\033[0m")

		var choice int
		_, err := fmt.Scan(&choice)
		if err != nil {
			fmt.Printf("\n\033[31mInvalid input. Please enter a number.\033[0m\n")

			fmt.Scanln()
			continue
		}

		switch choice {
		case 1:
			fmt.Println("\033[33m\nWarning:\033[0m")
			fmt.Println("\033[33m- A '.geojson' file with the farm name should be present in data/geojsons folder.\033[0m")
			fmt.Println("\033[33m- The '.geojson' file should contain the desired plot in its features identified by plot_id.\n\033[0m")
			reader := bufio.NewReader(os.Stdin)

			fmt.Print("\033[34mEnter the farm name: \033[0m")
			farm, _ := reader.ReadString('\n')
			farm = strings.TrimSpace(farm)

			fmt.Print("\033[34mEnter the plot id: \033[0m")
			plot, _ := reader.ReadString('\n')
			plot = strings.TrimSpace(plot)

			fmt.Print("\033[34mEnter the date to be analyzed (YYYY-MM-DD):  \033[0m")
			date, _ := reader.ReadString('\n')
			date = strings.TrimSpace(date)
			endDate, err := time.Parse("2006-01-02", date)
			if err != nil {
				fmt.Printf("\n\033[31mInvalid date format. Please use YYYY-MM-DD.\033[0m\n")
				continue
			}

			_, err = delivery.EvaluatePlot(farm, plot, endDate)
			if err != nil {
				fmt.Printf("\n\033[31mError evaluating plot: %s\033[0m\n", err.Error())
				continue
			}
		case 2:
			fmt.Println("\033[33m\nWarning:\033[0m")
			fmt.Println("\033[33mThe resultant dataset will be created at data/model folder\033[0m")
			fmt.Println("\033[33mThe input data should be a '.csv' file present in data/training_input folder\n\033[0m")

			fmt.Print("\033[34mEnter input data file name: \033[0m")
			var inputDataFileName string
			fmt.Scanln(&inputDataFileName)

			err := delivery.CreateDataset(inputDataFileName)
			if err != nil {
				fmt.Printf("\n\033[31mError creating dataset: %s\033[0m\n", err.Error())
				notification.SendDiscordErrorNotification(fmt.Sprintf("Maxsatt CLI\n\nError creating dataset: %s", err.Error()))
				continue
			}
			fmt.Printf("\n\033[32mDataset created successfully!\033[0m\n")
			notification.SendDiscordSuccessNotification(fmt.Sprintf("Maxsatt CLI\n\nDataset created successfully! \n\nFile: %s", inputDataFileName))
		case 3:
			println("Exiting...")
			return

		default:
			println("Invalid choice. Please try again.")
		}
	}
}

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		err := godotenv.Load("../.env")
		if err != nil {
			panic(err)
		}
	}

	initCLI()
}
