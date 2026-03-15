.PHONY: build run test tidy lint

build:
	go build -o vendor-dialogue-generator ./cmd/vendor-dialogue-generator/

run:
	go run ./cmd/vendor-dialogue-generator/

test:
	go test ./...

tidy:
	go mod tidy

lint:
	go vet ./...
