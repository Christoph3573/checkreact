.PHONY: run build seed generate tidy

run:
	go run ./cmd/server/main.go

build:
	go build -o bin/schulapp ./cmd/server/main.go

seed:
	go run ./seed/main.go

# requires: go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
generate:
	oapi-codegen -generate types,chi-server -package api -o internal/api/generated.go openapi.yaml

tidy:
	go mod tidy

test:
	go test ./...
