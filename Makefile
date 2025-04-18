.PHONY: all build docker clean fmt run deps vet lint test


export CGO_ENABLED=1

# Default target
all: build

# Build the binary
build:
	go build -o build/ ./cmd/nodeserver/... ./pkg/... ./vendor/...
	go build -o build/ ./cmd/nameserver/... ./pkg/... ./vendor/...

docker:
	docker build -t nodeserver:latest -f docker/nodeserver.dockerfile .
	docker build -t nameserver:latest -f docker/nameserver.dockerfile .

# Clean up build artifacts
clean:
	rm -f $(BINARY_NAME)

fmt:
	go fmt ./...

# Run the application
run: build
	./$(BINARY_NAME)

# Install dependencies
deps:
	go mod tidy
	go mod vendor

vet:
	go vet ./...

lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run

# Test the application
test:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

generate:
	go generate ./...