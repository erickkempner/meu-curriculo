.PHONY: run build docker templ

# Run locally
run:
	go run ./cmd/main.go

# Generate templ files
templ:
	templ generate

# Build Linux binary for Docker
build:
	templ generate
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build -o bin/server ./cmd/main.go

# Build Docker image
docker:
	docker compose build

# Run Docker
up:
	docker compose up --build

# Stop Docker
down:
	docker compose down
