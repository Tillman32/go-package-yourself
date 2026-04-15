BIN     := gpy
MAIN    := ./cmd/gpy

.PHONY: build test test-cover test-int lint clean install

build:
	go build -o $(BIN) $(MAIN)

test:
	go test -race $(shell go list ./... | grep -v /integration)

test-cover:
	go test -race -cover $(shell go list ./... | grep -v /integration)

test-int: build
	go test -race ./integration -v

lint:
	go vet ./...

clean:
	rm -f $(BIN)

install:
	go install $(MAIN)
