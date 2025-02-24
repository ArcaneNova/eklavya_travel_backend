FROM golang:1.21-alpine

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Create directory for SSL certificates
RUN mkdir -p /etc/ssl/certs

# Copy CA certificate
COPY certs/ca.crt /etc/ssl/certs/ca.crt

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o main .

# Make the entrypoint script executable
RUN chmod +x entrypoint.sh

# Expose port
EXPOSE 8080

# Run the application
CMD ["./entrypoint.sh"] 