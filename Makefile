all: build

build:
	go build -o doomsday cmd/*

clean:
	rm doomsday
