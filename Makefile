run:
	@trap 'kill 0' SIGINT; \
	python python-service/main.py --port=$(GRPC_PORT) & \
	cd go-service/cmd && clear && go run main.go --port=$(GRPC_PORT)

test-download:
	cd go-service/cmd/test_download && go run main.go