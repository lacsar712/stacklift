.PHONY: build test run clean

build:
	go build ./...

test:
	go test ./...

run:
	go run ./cmd/damgate

clean:
	go clean ./...
