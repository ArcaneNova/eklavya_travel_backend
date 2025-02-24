# Use the official Golang image as a parent image
FROM golang:1.21-alpine

# Set the working directory in the container
WORKDIR /app

# Install required system packages
RUN apk add --no-cache git postgresql-client bash

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN go build -o main .

# Create backup directory
RUN mkdir -p /app/backup

# Make scripts executable
RUN chmod +x /app/entrypoint.sh

# Expose port 8080
EXPOSE 8080

# Command to run the executable
CMD ["/app/entrypoint.sh"] 