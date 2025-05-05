package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/forest-guardian/forest-guardian-api-poc/internal/properties"
)

type DiscordMessage struct {
	Embeds []DiscordEmbed `json:"embeds"`
}

type DiscordEmbed struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Color       int    `json:"color"`
}

func SendDiscordErrorNotification(errorMessage string) error {
	message := DiscordMessage{
		Embeds: []DiscordEmbed{
			{
				Title:       "🚨 Error Notification",
				Description: fmt.Sprintf("So weird… must be your problem.\n\nAn error occurred: %s", errorMessage),
				Color:       16711680, // Red color
			},
		},
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	resp, err := http.Post(properties.DiscordErrorNotificationUrl(), "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send Discord notification, status code: %d", resp.StatusCode)
	}

	return nil
}

func SendDiscordWarnNotification(warnMessage string) error {
	message := DiscordMessage{
		Embeds: []DiscordEmbed{
			{
				Title:       "⚠️ Warning Notification",
				Description: fmt.Sprintf("Heads up!\n\n%s", warnMessage),
				Color:       16776960, // Yellow color
			},
		},
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	resp, err := http.Post(properties.DiscordErrorNotificationUrl(), "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send Discord notification, status code: %d", resp.StatusCode)
	}

	return nil
}

func SendDiscordSuccessNotification(successMessage string) error {
	message := DiscordMessage{
		Embeds: []DiscordEmbed{
			{
				Title:       "✅ Success Notification",
				Description: fmt.Sprintf("Not sure how, but it worked...\n\n%s", successMessage),
				Color:       65280, // Green color
			},
		},
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	resp, err := http.Post(properties.DiscordSuccessNotificationUrl(), "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send Discord notification, status code: %d", resp.StatusCode)
	}

	return nil
}
