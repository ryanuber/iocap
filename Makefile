test:
	go vet ./...
	go test -v -race ./...

.PHONY: test