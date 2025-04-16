# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/main.go

# Runtime stage
FROM alpine:latest

WORKDIR /

# Copy the binary from builder
COPY --from=builder /server /server

# Expose the port the app runs on
EXPOSE 8080

# Run the server
ENTRYPOINT ["/server"]