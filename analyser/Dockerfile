# Build
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set dir
WORKDIR /app

# Copy go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download


# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o analyser ./cmd/analyser

# Runtime
FROM alpine:3.20

# Install runtime dependencies
RUN apk add --no-cache ca-certificate wget

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set dir
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/analyser .

# Change ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose for future health check
EXPOSE 50051

# Run
CMD [ "./analyser" ]