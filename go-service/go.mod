module github.com/forest-guardian/forest-guardian-api-poc

go 1.24.0

require (
	github.com/airbusgeo/godal v0.0.13
	github.com/common-nighthawk/go-figure v0.0.0-20210622060536-734e95fb86be
	github.com/fatih/color v1.18.0
	github.com/fogleman/gg v1.3.0
	github.com/gammazero/workerpool v1.1.3
	github.com/gocarina/gocsv v0.0.0-20240520201108-78e41c74b4b1
	github.com/icza/mjpeg v0.0.0-20230330134156-38318e5ab8f4
	github.com/joho/godotenv v1.5.1
	github.com/paulmach/orb v0.11.1
	github.com/schollz/progressbar/v3 v3.18.0
	golang.org/x/oauth2 v0.28.0
	google.golang.org/grpc v1.72.0
	google.golang.org/protobuf v1.36.5
)

replace github.com/forest-guardian/forest-guardian-api-poc/internal/ml/protobufs => ./internal/ml/protobufs

replace github.com/forest-guardian/forest-guardian-api-poc/internal/delta/protobufs => ./internal/dataset/protobufs

require (
	github.com/gammazero/deque v0.2.0 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	go.mongodb.org/mongo-driver v1.11.4 // indirect
	golang.org/x/image v0.25.0 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/term v0.29.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
)
