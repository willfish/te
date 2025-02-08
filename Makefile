.PHONY: build test lint tidy

run: build
	./te

build:
	env GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -ldflags="-s -w" -o te cmd/te/main.go 

test: build
	go test ./...

lint: tidy
	golangci-lint run

tidy:
	go mod tidy
