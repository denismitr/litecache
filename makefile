.PHONY: deps
deps:
	go mod tidy

.PHONY: test
test:
	go test -race -count=1 ./...