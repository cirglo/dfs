# Use the official Golang image as the base image
FROM golang:1.24.1-alpine3.21 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files to the working directory
COPY go.mod go.sum ./

# Download the Go modules
RUN go mod download

# Copy the source code to the working directory
COPY . ./

RUN apk add --no-cache gcc musl-dev

# Build the Go application
RUN mkdir -p build
RUN CGO_ENABLED=1 GOOS=linux go build -o build/ ./cmd/nodeserver/... ./pkg/... ./vendor/...


FROM alpine:3.21.3 AS runtime

# Set the working directory inside the container
WORKDIR /app

# Copy the built executable from the builder stage
COPY --from=builder /app/build/nodeserver ./nodeserver

# Command to run the executable
CMD ["/app/nodeserver"]