package properties

import "os"

func RootPath() string {
	return os.Getenv("ROOT_PATH")
}

var GrpcPort int

type Color struct {
	R, G, B uint8
}

var ColorMap = map[string]Color{
	"unknown":              {128, 128, 128}, // gray
	"Psilideo":             {0, 255, 0},     // green
	"Formiga":              {255, 0, 0},     // red
	"Lagarta Desfolhadora": {0, 0, 255},     // blue
	"Saudavel":             {0, 0, 255},     // blue
}

func DiscordErrorNotificationUrl() string {
	return os.Getenv("DISCORD_ERROR_NOTIFICATION_URL")
}
func DiscordSuccessNotificationUrl() string {
	return os.Getenv("DISCORD_SUCCESS_NOTIFICATION_URL")
}
