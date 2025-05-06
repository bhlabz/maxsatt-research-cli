run:
	python python-service/main.py --port=$(GRPC_PORT) &
	cd go-service/cmd && clear && go run main.go --port=$(GRPC_PORT)
