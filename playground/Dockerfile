# Use the official Golang image with a specific version and targeting Linux as a base image
FROM golang:1.20-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go application dependency files
COPY go.mod go.sum ./

# Download and cache Go dependencies
RUN go mod download

# Copy the entire source code into the container
COPY . .

# Build the Go application for Linux
RUN CGO_ENABLED=0 GOOS=linux go build -o re-redis-playground

# Use a minimal Alpine Linux image for the final image
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /app

# Copy only the built executable from the previous stage
COPY --from=builder /app/re-redis-playground /app/re-redis-playground

# Command to run the executable
CMD ["./re-redis-playground"]
