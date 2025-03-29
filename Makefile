.PHONY: all build clean run deps test

# Default target
all: build

# Build the binary
build:
	go build -o build ./cmd/nodeserver/... ./pkg/... ./vendor/...

# Clean up build artifacts
clean:
	rm -f $(BINARY_NAME)

# Run the application
run: build
	./$(BINARY_NAME)

# Install dependencies
deps:
	go mod tidy
	go mod vendor

# Test the application
test:
	go test ./...

generate:
	go generate ./...