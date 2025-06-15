# Build stage
FROM golang:1.21-alpine AS builder

# Install git (needed for version info)
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mdnotes ./cmd

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates git

# Create a non-root user
RUN addgroup -g 1001 -S mdnotes && \
    adduser -S mdnotes -u 1001 -G mdnotes

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/mdnotes .

# Copy documentation
COPY --from=builder /app/docs ./docs
COPY --from=builder /app/README.md ./
COPY --from=builder /app/LICENSE ./

# Change ownership to mdnotes user
RUN chown -R mdnotes:mdnotes /app

# Switch to non-root user
USER mdnotes

# Set the binary as the entrypoint
ENTRYPOINT ["./mdnotes"]

# Default command
CMD ["--help"]

# Metadata
LABEL org.opencontainers.image.title="mdnotes"
LABEL org.opencontainers.image.description="A powerful CLI tool for managing Obsidian markdown note vaults"
LABEL org.opencontainers.image.url="https://github.com/eoinhurrell/mdnotes"
LABEL org.opencontainers.image.source="https://github.com/eoinhurrell/mdnotes"
LABEL org.opencontainers.image.licenses="MIT"