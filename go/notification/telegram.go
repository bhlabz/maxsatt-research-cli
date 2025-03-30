package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

var (
	token  = "your_bot_token" // Replace with your bot's token
	chatID = "your_chat_id"   // Replace with your chat ID
)

// sendTelegramMessage sends a message to Telegram.
func sendTelegramMessage(message string, parseMode *string) (map[string]interface{}, error) {
	message = escapeExclamationMark(message)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    message,
	}
	if parseMode != nil {
		payload["parse_mode"] = *parseMode
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	fmt.Printf("Telegram Response Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Telegram Response Content: %v\n", response)
	return response, nil
}

// sendErrorMessage sends an error message with Exception details to Telegram using Markdown.
func sendErrorMessage(exception error, message string) error {
	errorMessage := fmt.Sprintf(
		"*🚨 Error Occurred!* \n\n"+
			"*Type:* `%T`\n"+
			"*Message:* `%s`\n"+
			"*Error Message:* `%s`",
		exception, message, exception.Error(),
	)
	parseMode := "MarkdownV2"
	_, err := sendTelegramMessage(errorMessage, &parseMode)
	return err
}

// sendSuccessMessage sends a success message to Telegram using Markdown.
func sendSuccessMessage(message string) error {
	successMessage := fmt.Sprintf("*✅ Success!* \n\n%s", message)
	parseMode := "MarkdownV2"
	_, err := sendTelegramMessage(successMessage, &parseMode)
	return err
}

// sendWarnMessage sends a warning message to Telegram using Markdown.
func sendWarnMessage(message string) error {
	warnMessage := fmt.Sprintf("*⚠️ Warning!* \n\n%s", message)
	parseMode := "MarkdownV2"
	_, err := sendTelegramMessage(warnMessage, &parseMode)
	return err
}

// escapeExclamationMark escapes exclamation marks in the message for MarkdownV2.
func escapeExclamationMark(message string) string {
	return string(bytes.ReplaceAll([]byte(message), []byte("!"), []byte("\\!")))
}
