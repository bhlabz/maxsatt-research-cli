run:
	python python-service/main.py &
	cd go-service/cmd && clear && go run main.go