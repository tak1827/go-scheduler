lint:
	@echo "--> Running linter"
	@go vet ./...
	@golangci-lint run --timeout=10m

fmt:
	@gofmt -w -s -l .
	@misspell -w .

test:
	go test ./... -race

bench:
	go test ./... -bench=. -benchtime=2s -benchmem
