# Stage 1: Build the Go binary
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build statically linked binary (CGO disabled for pure Go SQLite)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./cmd/server

# Stage 2: Ultra-lightweight runtime container
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server /app/server

# Create data directory for SQLite persistence
RUN mkdir -p /app/data

ENV PORT=8000
ENV DB_PATH=/app/data/lovenest.db

EXPOSE 8000

CMD ["/app/server"]
