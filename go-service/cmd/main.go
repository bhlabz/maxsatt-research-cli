package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	_ "image/png" // support PNG decoding

	"github.com/common-nighthawk/go-figure"
	"github.com/fatih/color"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/delivery"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/notification"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
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
			// Get the function, file, and line where panic occurred
			pc, file, line, ok := runtime.Caller(3) // 3 levels up is often the panic source
			var location string
			if ok {
				fn := runtime.FuncForPC(pc)
				location = fmt.Sprintf("%s:%d in %s", file, line, fn.Name())
			} else {
				location = "Unknown location"
			}

			// Print structured error
			fmt.Printf("\n\033[31mPANIC: %v\033[0m\n", r)
			fmt.Printf("\033[31mLocation: %s\033[0m\n", location)
			fmt.Printf("\033[31mPlease check the input and try again.\033[0m\n")
			fmt.Printf("\033[31mExiting...\033[0m\n")

			// Prepare full error message
			stack := debug.Stack()
			errMessage := fmt.Sprintf("Maxsatt CLI panic:\n\n%v\n\nLocation: %s\n\nStack trace:\n%s", r, location, stack)
			err := notification.SendDiscordErrorNotification(errMessage)
			if err != nil {
				fmt.Printf("\033[31mFailed to send notification: %s\033[0m\n", err.Error())
			}
		}
	}()
	printBanner()

	for {
		fmt.Println("\033[34m===================\033[0m")
		fmt.Println("\033[34m1. Evaluate a forest plot\033[0m")
		fmt.Println("\033[34m2. Create a new dataset\033[0m")
		fmt.Println("\033[34m3. List available forests\033[0m")
		fmt.Println("\033[34m4. List available forest plots\033[0m")
		fmt.Println("\033[34m5. Exit\033[0m")
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

			fmt.Print("\033[34mEnter the forest name: \033[0m")
			forest, _ := reader.ReadString('\n')
			forest = strings.TrimSpace(forest)

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

			path, err := delivery.EvaluatePlot(forest, plot, endDate)
			if err != nil {
				fmt.Printf("\n\033[31mError evaluating plot: %s\033[0m\n", err.Error())
				continue
			}
			fmt.Printf("\n\033[32mSuccessful analysis! Resultant image located at: %s\033[0m\n", path)
		case 2:
			fmt.Println("\033[33m\nWarning:\033[0m")
			fmt.Println("\033[33mThe resultant dataset will be created at data/model folder\033[0m")
			fmt.Println("\033[33mThe input data should be a '.csv' file present in data/training_input folder\n\033[0m")

			fmt.Print("\033[34mEnter input data file name: \033[0m")
			var inputDataFileName string
			fmt.Scanln(&inputDataFileName)

			fmt.Print("\033[34mEnter the ideal delta days for the image analysis: \033[0m")
			var deltaDays int
			fmt.Scanln(&deltaDays)

			fmt.Print("\033[34mEnter the delta days trash hold for the image analysis: \033[0m")
			var deltaDaysTrashHold int
			fmt.Scanln(&deltaDaysTrashHold)

			outputtDataFileName := fmt.Sprintf("%s_%s_%d_%d.csv", strings.TrimSuffix(inputDataFileName, ".csv"), time.Now().Format("2006-01-02"), deltaDays, deltaDaysTrashHold)
			err := delivery.CreateDataset(inputDataFileName, outputtDataFileName, deltaDays, deltaDaysTrashHold)
			if err != nil {
				fmt.Printf("\n\033[31mError creating dataset: %s\033[0m\n", err.Error())
				if !strings.Contains(err.Error(), "empty csv file given") {
					notification.SendDiscordErrorNotification(fmt.Sprintf("Maxsatt CLI\n\nError creating dataset: %s", err.Error()))
				}
				continue
			}
			fmt.Printf("\n\033[32mDataset created successfully!\033[0m\n")
			notification.SendDiscordSuccessNotification(fmt.Sprintf("Maxsatt CLI\n\nDataset created successfully! \n\nFile: %s", inputDataFileName))
		case 3:
			files, err := os.ReadDir(properties.RootPath() + "/data/geojsons")
			if err != nil {
				fmt.Printf("\n\033[31mError reading geojsons folder: %s\033[0m\n", err.Error())
				return
			}
			fmt.Println("\033[33m\nWarning:\033[0m")
			fmt.Println("\033[33mTo add a new forest, add its '.geojson' file at 'data/geojsons' folder.\033[0m")

			fmt.Println("\n\033[32mAvailable forests:\033[0m")
			for _, file := range files {
				if strings.HasSuffix(file.Name(), ".geojson") {
					fmt.Printf("\033[32m- %s\033[0m\n", strings.TrimSuffix(file.Name(), ".geojson"))
				}
			}

		case 4:
			fmt.Println("\033[33m\nWarning:\033[0m")
			fmt.Println("\033[33mTo add a plot to a forest add the 'plot_id' property at the '.geojson' file from the forest fo your choice.\033[0m")
			fmt.Println("\033[33mThe 'plot_id' property should be located at 'features[N]properties.plot_id'.\n\033[0m")

			reader := bufio.NewReader(os.Stdin)

			fmt.Print("\033[34mEnter the forest name: \033[0m")
			forest, _ := reader.ReadString('\n')
			forest = strings.TrimSpace(forest)
			filePath := properties.RootPath() + "/data/geojsons/" + forest + ".geojson"
			file, err := os.Open(filePath)
			if err != nil {
				fmt.Printf("\n\033[31mError opening file: %s\033[0m\n", err.Error())
				continue
			}
			defer file.Close()

			var geojsonData map[string]interface{}
			decoder := json.NewDecoder(file)
			err = decoder.Decode(&geojsonData)
			if err != nil {
				fmt.Printf("\n\033[31mError decoding GEOJSON: %s\033[0m\n", err.Error())
				continue
			}

			features, ok := geojsonData["features"].([]interface{})
			if !ok {
				fmt.Printf("\n\033[31mError: Invalid GEOJSON format.\033[0m\n")
				continue
			}

			plotIDs := []string{}

			for _, feature := range features {
				featureMap, ok := feature.(map[string]interface{})
				if !ok {
					fmt.Printf("\n\033[31mError: Invalid feature format.\033[0m\n")
					break
				}
				properties, ok := featureMap["properties"].(map[string]interface{})
				if !ok {
					fmt.Printf("\n\033[31mError: Invalid properties format.\033[0m\n")
					break
				}
				plotID, ok := properties["plot_id"].(string)
				if ok {
					plotIDs = append(plotIDs, plotID)
				}
			}
			if len(plotIDs) == 0 {
				fmt.Printf("\n\033[31mNo plot IDs found in the GEOJSON file.\033[0m\n")
				continue
			}
			fmt.Println("\033[32m\nAvailable plots:\033[0m")
			for _, plotID := range plotIDs {
				fmt.Printf("\033[32m- %s\033[0m\n", plotID)
			}
		case 5:
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
