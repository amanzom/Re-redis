# Use the official Golang image with a specific version and targeting Linux as a base image
FROM golang:1.20-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy all files into the container
COPY . .

# Copy only the necessary Go application dependency files to maximize caching
COPY go.mod go.sum ./

# Download and cache Go dependencies
RUN go mod download

# Build the Go application for Linux
RUN CGO_ENABLED=0 GOOS=linux go build -o re-redis

# Use a minimal Alpine Linux image for the final image
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /app

# Copy all files, including the built executable and additional files, from the previous stage
COPY --from=builder /app /app

# Command to run the executable
CMD ["./re-redis"]
