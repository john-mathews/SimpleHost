# Use official Golang image
FROM golang:1.24-alpine

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the app
COPY . .

# Install build tools for CGO/SQLite
RUN apk add --no-cache gcc musl-dev

# Build the Go app with CGO enabled
ENV CGO_ENABLED=1
RUN go build -o server .

RUN ls -l /app

# Expose port 8080
EXPOSE 8080

# Run the server
CMD ["/app/server"]
