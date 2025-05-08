package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/common-nighthawk/go-figure"
	bannercolor "github.com/fatih/color"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/delivery"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/notification"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
	"github.com/forest-guardian/forest-guardian-api-poc/output"
	"github.com/joho/godotenv"
)

func printBanner() {
	// Print the banner with go-figure
	figure1 := figure.NewFigure("Maxsatt", "isometric1", true)
	figure2 := figure.NewFigure("CLI", "isometric1", true)
	bannercolor.Cyan(figure1.String())
	bannercolor.Cyan(figure2.String())
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
		fmt.Println("\033[34m2. Evaluate a forest plot image indexes\033[0m")
		fmt.Println("\033[34m3. Create a new dataset\033[0m")
		fmt.Println("\033[34m4. List available forests\033[0m")
		fmt.Println("\033[34m5. List available forest plots\033[0m")
		fmt.Println("\033[34m6. Exit\033[0m")
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

			modelFolderPath := fmt.Sprintf("%s/data/model/", properties.RootPath())

			modelFiles, err := os.ReadDir(modelFolderPath)
			if err != nil {
				fmt.Printf("\n\033[31mError reading model folder: %s\033[0m\n", err.Error())
				continue
			}

			if len(modelFiles) == 0 {
				fmt.Printf("\n\033[31mNo models found in the model folder.\033[0m\n")
				continue
			}

			fmt.Println("\033[32m\nAvailable models:\033[0m")
			for i, file := range modelFiles {
				fmt.Printf("\033[32m%d. %s\033[0m\n", i+1, file.Name())
			}

			fmt.Print("\033[34mEnter the number of the model you want to use: \033[0m")
			var modelChoice int
			_, err = fmt.Scan(&modelChoice)
			if err != nil || modelChoice < 1 || modelChoice > len(modelFiles) {
				fmt.Printf("\n\033[31mInvalid choice. Please select a valid model number.\033[0m\n")
				continue
			}

			selectedModel := modelFiles[modelChoice-1].Name()
			fmt.Printf("\033[32mYou selected the model: %s\033[0m\n", selectedModel)

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
				continue
			}

			result, err := delivery.EvaluatePlotFinalData(selectedModel, forest, plot, endDate)
			if err != nil {
				fmt.Printf("\n\033[31mError evaluating plot: %s\033[0m\n", err.Error())
				if !strings.Contains(err.Error(), "empty csv file given") {
					// notification.SendDiscordErrorNotification(fmt.Sprintf("Maxsatt CLI\n\nError evaluating plot: %s", err.Error()))
				}
				continue
			}

			imageFolderPath := fmt.Sprintf("%s/data/images/%s_%s/", properties.RootPath(), forest, plot)

			files, err := os.ReadDir(imageFolderPath)
			if err != nil {
				fmt.Printf("\n\033[31mError reading image folder: %s\033[0m\n", err.Error())
				// notification.SendDiscordErrorNotification(fmt.Sprintf("Maxsatt CLI\n\nError reading images folder: %s", err.Error()))
				continue
			}

			if len(files) == 0 {
				fmt.Printf("\n\033[31mNo tiff images found to create resultant image\033[0m\n")
				// notification.SendDiscordErrorNotification("Maxsatt CLI\n\nNo tiff images found to create resultant image")
				continue
			}

			firstFileName := files[0].Name()
			firstFilePath := fmt.Sprintf("%s%s", imageFolderPath, firstFileName)

			outputFileName := fmt.Sprintf("%s_%s_%s", forest, plot, endDate.Format("2006-01-02"))

			outputGeoJsonFilePath := output.CreateFinalDataGeoJson(result, outputFileName)

			outputImageFilePath, err := output.CreateFinalDataImage(result, firstFilePath, outputFileName)
			if err != nil {
				fmt.Printf("\n\033[31mError creating resultant image: %s\033[0m\n", err.Error())
				// notification.SendDiscordErrorNotification(fmt.Sprintf("Maxsatt CLI\n\nError creating resultant image: %s", err.Error()))
				continue
			}

			fmt.Printf("\n\033[32mSuccessful analysis!\n Resultant image located at: %s\n Resultant geojson located at: %s\033[0m\n", outputImageFilePath, outputGeoJsonFilePath)
			// notification.SendDiscordSuccessNotification(fmt.Sprintf("Maxsatt CLI\n\nSuccessful analysis!\nResultant image located at: %s\nResultant geojson located at: %s", outputImageFilePath, outputGeoJsonFilePath))
		case 2:
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

			fmt.Print("\033[34mEnter the end date (YYYY-MM-DD): \033[0m")
			endDateInput, _ := reader.ReadString('\n')
			endDateInput = strings.TrimSpace(endDateInput)
			endDate, err := time.Parse("2006-01-02", endDateInput)
			if err != nil {
				fmt.Printf("\n\033[31mInvalid date format: %s. Please use YYYY-MM-DD.\033[0m\n", endDateInput)
				continue
			}

			fmt.Print("\033[34mEnter interval in days: \033[0m")
			intervalInput, _ := reader.ReadString('\n')
			intervalInput = strings.TrimSpace(intervalInput)
			intervalDays, err := strconv.Atoi(intervalInput)
			if err != nil || intervalDays <= 0 {
				fmt.Printf("\n\033[31mInvalid interval: %s. Please enter a positive integer.\033[0m\n", intervalInput)
				continue
			}

			fmt.Print("\033[34mEnter number of samples: \033[0m")
			samplesInput, _ := reader.ReadString('\n')
			samplesInput = strings.TrimSpace(samplesInput)
			samples, err := strconv.Atoi(samplesInput)
			if err != nil || samples <= 0 {
				fmt.Printf("\n\033[31mInvalid number of samples: %s. Please enter a positive integer.\033[0m\n", samplesInput)
				continue
			}

			var endDates []time.Time
			for i := range samples {
				endDates = append(endDates, endDate.AddDate(0, 0, -i*intervalDays))
			}

			sort.Slice(endDates, func(i, j int) bool {
				return endDates[i].Before(endDates[j])
			})

			resultPath := fmt.Sprintf("%s/data/result/%s/%s", properties.RootPath(), forest, plot)

			err = os.MkdirAll(resultPath, os.ModePerm)
			if err != nil {
				log.Fatalf("Failed to create result folder: %v", err)
			}

			var outputImageFilePaths []string

			for _, endDate := range endDates {

				result, err := delivery.EvaluatePlotCleanData(forest, plot, endDate)
				if err != nil {
					fmt.Printf("\n\033[31mError evaluating plot: %s\033[0m\n", err.Error())
					if !strings.Contains(err.Error(), "empty csv file given") {
						// notification.SendDiscordErrorNotification(fmt.Sprintf("Maxsatt CLI\n\nError evaluating plot: %s", err.Error()))
					}
					continue
				}

				imageFolderPath := fmt.Sprintf("%s/data/images/%s_%s/", properties.RootPath(), forest, plot)

				files, err := os.ReadDir(imageFolderPath)
				if err != nil {
					fmt.Printf("\n\033[31mError reading image folder: %s\033[0m\n", err.Error())
					// notification.SendDiscordErrorNotification(fmt.Sprintf("Maxsatt CLI\n\nError reading images folder: %s", err.Error()))
					continue
				}

				if len(files) == 0 {
					fmt.Printf("\n\033[31mNo tiff images found to create resultant image\033[0m\n")
					// notification.SendDiscordErrorNotification("Maxsatt CLI\n\nNo tiff images found to create resultant image")
					continue
				}

				firstFileName := files[0].Name()
				firstFilePath := fmt.Sprintf("%s%s", imageFolderPath, firstFileName)

				outputFilePath := fmt.Sprintf("%s/%s_%s_%s", resultPath, forest, plot, endDate.Format("2006-01-02"))

				outputImageFilePath, err := output.CreateCleanDataImage(result, firstFilePath, outputFilePath)
				if err != nil {
					fmt.Printf("\n\033[31mError creating resultant image: %s\033[0m\n", err.Error())
					// notification.SendDiscordErrorNotification(fmt.Sprintf("Maxsatt CLI\n\nError creating resultant image: %s", err.Error()))
					continue
				}

				fmt.Printf("\n\033[32mSuccessful analysis!\n Resultant image located at: %s\033[0m\n", outputImageFilePath)
				outputImageFilePaths = append(outputImageFilePaths, outputImageFilePath)
			}

			if len(outputImageFilePaths) > 1 {
				outputVideoPath := fmt.Sprintf("%s/%s_%s_%s_%d_%d", resultPath, forest, plot, endDate.Format("2006-01-02"), samples, intervalDays)
				output.CreateVideoFromImages(outputImageFilePaths, outputVideoPath)
			}

		case 3:
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
			var deltaDaysThreshold int
			fmt.Scanln(&deltaDaysThreshold)

			outputtDataFileName := fmt.Sprintf("%s_%s_%d_%d.csv", strings.TrimSuffix(inputDataFileName, ".csv"), time.Now().Format("2006-01-02"), deltaDays, deltaDaysThreshold)
			err := delivery.CreateDataset(inputDataFileName, outputtDataFileName, deltaDays, deltaDaysThreshold)
			if err != nil {
				fmt.Printf("\n\033[31mError creating dataset: %s\033[0m\n", err.Error())
				if !strings.Contains(err.Error(), "empty csv file given") {
					notification.SendDiscordErrorNotification(fmt.Sprintf("Maxsatt CLI\n\nError creating dataset: %s", err.Error()))
				}
				continue
			}
			fmt.Printf("\n\033[32mDataset created successfully!\033[0m\n")
			notification.SendDiscordSuccessNotification(fmt.Sprintf("Maxsatt CLI\n\nDataset created successfully! \n\nFile: %s", inputDataFileName))
		case 4:
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

		case 5:
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
		case 6:
			println("Exiting...")
			return
		default:
			println("Invalid choice. Please try again.")
		}
	}
}

func main() {
	var port int
	for i, arg := range os.Args {
		if strings.HasPrefix(arg, "--port=") {
			portArg := strings.TrimPrefix(arg, "--port=")
			var err error
			port, err = strconv.Atoi(portArg)
			if err != nil {
				fmt.Printf("\033[31mInvalid port value: %s\033[0m\n", portArg)
				os.Exit(1)
			}
			break
		} else if arg == "--port" && i+1 < len(os.Args) {
			var err error
			port, err = strconv.Atoi(os.Args[i+1])
			if err != nil {
				fmt.Printf("\033[31mInvalid port value: %s\033[0m\n", os.Args[i+1])
				os.Exit(1)
			}
			break
		}
	}

	if port == 0 {
		port = 50051
		fmt.Printf("\033[33mNo port specified. Using default port: %d\033[0m\n", port)
	} else {
		fmt.Printf("\033[32mUsing specified port: %d\033[0m\n", port)
	}

	err := godotenv.Load("../../.env")
	if err != nil {
		err := godotenv.Load("../.env")
		if err != nil {
			panic(err)
		}
	}

	properties.GrpcPort = port
	initCLI()
}
