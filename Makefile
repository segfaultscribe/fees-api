.PHONY: fmt lint test tidy build run check

fmt:
	gofumpt -l -w .

lint:
	golangci-lint run ./...

test:
	go test -v -race ./...

tidy:
	go mod tidy
	go mod verify

build:
	go build -o bin/api

run:
	encore run

check: fmt lint test