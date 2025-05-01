package properties

const RootPath = "/Users/gabihert/Projects/forest-guardian/forest-guardian-api-poc/go"

type Color struct {
	R, G, B uint8
}

var ColorMap = map[string]Color{
	"unknown": {255, 0, 0},
}
