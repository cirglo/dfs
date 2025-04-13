.PHONY: all build docker clean fmt run deps vet lint test

# Detect the operating system using ComSpec for Windows
ifeq ($(ComSpec),)
    SEP := /
else
    SEP := \\
endif

export CGO_ENABLED=1

# Default target
all: build

# Build the binary
build:
	go build -o build .$(SEP)cmd$(SEP)nodeserver$(SEP)... .$(SEP)pkg$(SEP)... .$(SEP)vendor$(SEP)...
	go build -o build .$(SEP)cmd$(SEP)nameserver$(SEP)... .$(SEP)pkg$(SEP)... .$(SEP)vendor$(SEP)...

docker:
	docker build -t nodeserver:latest -f docker$(SEP)nodeserver.dockerfile .

# Clean up build artifacts
clean:
	rm -f $(BINARY_NAME)

fmt:
	go fmt .$(SEP)...

# Install dependencies
deps:
	go mod tidy
	go mod vendor

vet:
	go vet .$(SEP)...

lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run

# Test the application
test:
	go test .$(SEP)...

generate:
	go generate .$(SEP)...