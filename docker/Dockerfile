# Multi-stage build für Go Backend
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY backend/go.mod backend/go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY backend/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage with Nginx
FROM nginx:alpine

# Install ca-certificates for API calls
RUN apk --no-cache add ca-certificates

# Copy Go binary (gws.md is now embedded in the binary)
COPY --from=builder /app/main /app/main

# Copy frontend files
COPY frontend/ /usr/share/nginx/html/

# Copy nginx configuration
COPY docker/nginx-combined.conf /etc/nginx/conf.d/default.conf

# Copy startup script
COPY docker/start-go.sh /start.sh
RUN chmod +x /start.sh

EXPOSE 80

CMD ["/start.sh"]
