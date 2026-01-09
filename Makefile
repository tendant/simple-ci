.PHONY: build run test fmt lint clean docker-build

# Build the gateway binary
build:
	go build -o bin/gateway ./cmd/gateway

# Run the gateway locally
run:
	go run ./cmd/gateway

# Run tests
test:
	go test -v -race ./...

# Run tests with coverage
coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Format code
fmt:
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Tidy dependencies
tidy:
	go mod tidy

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out

# Build Docker image
docker-build:
	docker build -t simple-ci-gateway:latest .

# Run Docker container
docker-run:
	docker run -p 8080:8080 \
		-v $(PWD)/configs:/app/configs \
		simple-ci-gateway:latest
