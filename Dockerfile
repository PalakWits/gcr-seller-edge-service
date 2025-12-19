FROM golang:1.24-alpine

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . ./

# Build the application
RUN go build -o adapter cmd/main.go

# Expose port
EXPOSE 8083

CMD ["./adapter"]