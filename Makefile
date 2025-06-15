.PHONY: build test clean install lint fmt vet deps release snapshot docker

# Build the binary
build:
	go build -o mdnotes ./cmd

# Run tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -f mdnotes coverage.out coverage.html

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Install the binary
install:
	go install ./cmd

# Run all checks
check: fmt vet test

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o bin/mdnotes-linux-amd64 ./cmd
	GOOS=darwin GOARCH=amd64 go build -o bin/mdnotes-darwin-amd64 ./cmd
	GOOS=windows GOARCH=amd64 go build -o bin/mdnotes-windows-amd64.exe ./cmd

# Development build with race detection
dev-build:
	go build -race -o mdnotes ./cmd

# Run benchmarks
bench:
	go test -bench=. ./...

# Release with goreleaser
release:
	goreleaser release --clean

# Create a snapshot release (no tags required)
snapshot:
	goreleaser release --snapshot --clean

# Build Docker image
docker:
	docker build -t mdnotes:latest .

# Run Docker container
docker-run:
	docker run --rm -v $(PWD):/vault mdnotes:latest analyze stats /vault

# Install goreleaser (for development)
install-goreleaser:
	go install github.com/goreleaser/goreleaser@latest

# Pre-release checks
pre-release: clean test bench lint
	@echo "All checks passed. Ready for release!"

# Development setup
setup: deps install-goreleaser
	@echo "Development environment setup complete"