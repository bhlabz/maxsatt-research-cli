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
	"unknown": {255, 0, 0},
}

func DiscordErrorNotificationUrl() string {
	return os.Getenv("DISCORD_ERROR_NOTIFICATION_URL")
}
func DiscordSuccessNotificationUrl() string {
	return os.Getenv("DISCORD_SUCCESS_NOTIFICATION_URL")
}
