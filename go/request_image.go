package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2/clientcredentials"
)

func requestImage(startDate, endDate string, geometry map[string]any, widthPixels, heightPixels int) ([]byte, error) {
	evalscript := `
    //VERSION=3
    function setup() {
      return {
        input: ["B05", "B08", "B11", "B02", "B04", "B06", "CLD", "SCL"],
        output: {
          id: "default",
          bands: 8,
          sampleType: SampleType.FLOAT32,
        },
      }
    }

    function evaluatePixel(sample) {
      return [sample.B05, sample.B08, sample.B11, sample.B02, sample.B04, sample.B06, sample.CLD, sample.SCL];
    }
  `

	requestPayload := map[string]interface{}{
		"input": map[string]interface{}{
			"bounds": map[string]interface{}{
				"geometry": geometry,
			},
			"data": []map[string]interface{}{
				{
					"dataFilter": map[string]interface{}{
						"timeRange": map[string]string{
							"from": startDate,
							"to":   endDate,
						},
					},
					"type": "sentinel-2-l2a",
				},
			},
		},
		"output": map[string]interface{}{
			"width":  heightPixels,
			"height": widthPixels,
			"responses": []map[string]interface{}{
				{
					"identifier": "default",
					"format": map[string]string{
						"type": "image/tiff",
					},
				},
			},
		},
		"evalscript": evalscript,
		"mosaicking": "mostRecent",
	}

	// Serialize the request payload to JSON
	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %v", err)
	}

	// OAuth2 configuration from environment variables
	clientID := os.Getenv("COPERNICUS_CLIENT_ID")
	clientSecret := os.Getenv("COPERNICUS_CLIENT_SECRET")
	tokenURL := os.Getenv("COPERNICUS_TOKEN_URL")

	if clientID == "" || clientSecret == "" || tokenURL == "" {
		return nil, fmt.Errorf("missing required environment variables: COPERNICUS_CLIENT_ID, COPERNICUS_CLIENT_SECRET, or COPERNICUS_TOKEN_URL")
	}

	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
	}

	// Create an HTTP client with OAuth2
	httpClient := config.Client(context.Background())

	url := "https://sh.dataspace.copernicus.eu/api/v1/process"

	// Retry logic
	retries := 3
	var response *http.Response
	for attempt := 1; attempt <= retries; attempt++ {
		response, err = httpClient.Post(url, "application/json", bytes.NewBuffer(requestBody))
		if err == nil && response.StatusCode == http.StatusOK {
			break
		}

		if response != nil {
			body, _ := ioutil.ReadAll(response.Body)
			fmt.Printf("Attempt %d failed: %s\n", attempt, string(body))
			response.Body.Close()
		} else {
			fmt.Printf("Attempt %d failed: %v\n", attempt, err)
		}

		time.Sleep(2 * time.Second) // Wait for 2 seconds before retrying
	}

	if err != nil {
		return nil, fmt.Errorf("failed to request image after %d attempts: %v", retries, err)
	}
	defer response.Body.Close()

	// Read the response body
	responseContent, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	return responseContent, nil
}
