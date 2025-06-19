.PHONY: build test clean install lint fmt vet deps release snapshot docker completions install-completions

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

# Generate shell completion scripts
completions: build
	@mkdir -p completions
	./mdnotes completion bash > completions/mdnotes.bash
	./mdnotes completion zsh > completions/_mdnotes
	./mdnotes completion fish > completions/mdnotes.fish
	./mdnotes completion powershell > completions/mdnotes.ps1
	@echo "Shell completion scripts generated in completions/ directory"

# Install shell completions (requires sudo for system-wide install)
install-completions: completions
	@echo "Installing shell completions..."
	@if command -v bash >/dev/null 2>&1; then \
		if [ -d /etc/bash_completion.d ]; then \
			sudo cp completions/mdnotes.bash /etc/bash_completion.d/mdnotes; \
			echo "Bash completion installed to /etc/bash_completion.d/"; \
		elif [ -d /usr/local/etc/bash_completion.d ]; then \
			sudo cp completions/mdnotes.bash /usr/local/etc/bash_completion.d/mdnotes; \
			echo "Bash completion installed to /usr/local/etc/bash_completion.d/"; \
		else \
			echo "Bash completion directory not found. Copy completions/mdnotes.bash manually."; \
		fi; \
	fi
	@if command -v zsh >/dev/null 2>&1; then \
		if [ -d /usr/local/share/zsh/site-functions ]; then \
			sudo cp completions/_mdnotes /usr/local/share/zsh/site-functions/; \
			echo "Zsh completion installed to /usr/local/share/zsh/site-functions/"; \
		else \
			echo "Zsh completion directory not found. Copy completions/_mdnotes to your fpath."; \
		fi; \
	fi
	@if command -v fish >/dev/null 2>&1; then \
		if [ -d ~/.config/fish/completions ]; then \
			cp completions/mdnotes.fish ~/.config/fish/completions/; \
			echo "Fish completion installed to ~/.config/fish/completions/"; \
		else \
			echo "Fish completion directory not found. Create ~/.config/fish/completions/ first."; \
		fi; \
	fi
	@echo "Installation complete! Restart your shell or source the completion files."

# Development setup
setup: deps install-goreleaser
	@echo "Development environment setup complete"