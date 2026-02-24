.PHONY: lint fmt vet test update

# Format code (fixes style issues)
fmt:
	go fmt ./...
	gofmt -s -w .

# Static analysis
vet:
	go vet ./...

# Run tests
test:
	go test ./...
	
# Run formatting and vet checks
lint: vet
	golangci-lint run -E gocyclo -E misspell

update:
	git pull
	go mod tidy