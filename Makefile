.PHONY: build test lint tidy run-parse run-browse

build:
	env GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -ldflags="-s -w" -o te ./cmd/te/

test: build
	go test ./...

lint: tidy
	golangci-lint run

tidy:
	go mod tidy

run-parse: build
	./te parse $(FILE)

run-browse: build
	./te browse
