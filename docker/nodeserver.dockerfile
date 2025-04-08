# Use the official Golang image as the base image
FROM golang:1.24.1-alpine3.21

# Set the working directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files to the working directory
COPY go.mod go.sum ./

# Download the Go modules
RUN go mod download

# Copy the source code to the working directory
COPY . ./

# Build the Go application
RUN mkdir -p build
RUN CGO_ENABLED=0 GOOS=linux go build -o build ./cmd/nodeserver/... ./pkg/... ./vendor/...

# Expose the port that the application will run on
EXPOSE 8080
EXPOSE 5001

# Command to run the executable
CMD ["/nodeserver"]