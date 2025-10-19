# ---- Build stage ----
FROM golang:1.24 AS builder

WORKDIR /app

# Copy go.mod and go.sum
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy all backend source files
COPY backend/ ./

# Build the Go binary with explicit output path
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/server .

# ---- Runtime stage ----
FROM debian:bookworm-slim

# Install SQLite and CA certificates
RUN apt-get update && \
    apt-get install -y sqlite3 ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the compiled binary from builder
COPY --from=builder /app/server .

# Copy migrations directory from builder
COPY --from=builder /app/pkg/db/migrations/ ./pkg/db/migrations/

# Create uploads directory and copy if exists
RUN mkdir -p ./uploads
COPY --from=builder /app/uploads/ ./uploads/

# Make sure the binary is executable
RUN chmod +x /app/server

# Create data directory
RUN mkdir -p /app/data

# Ensure uploads directory has proper permissions
RUN chmod -R 755 /app/uploads

EXPOSE 8088

CMD ["/app/server"]