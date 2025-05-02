run:
	python python-service/main.py &
	cd go-service/cmd && go run main.go