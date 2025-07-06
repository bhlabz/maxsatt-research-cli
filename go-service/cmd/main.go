package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/common-nighthawk/go-figure"
	bannercolor "github.com/fatih/color"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/notification"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/ui"
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

	// Use the new UI package to show the menu
	ui.ShowMenu()
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
