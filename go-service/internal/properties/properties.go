package properties

import "os"

func RootPath() string {
	return os.Getenv("ROOT_PATH")
}

type Color struct {
	R, G, B uint8
}

var ColorMap = map[string]Color{
	"unknown": {255, 0, 0},
}
