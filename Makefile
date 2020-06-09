all: generate build

generate:
	go generate ./...

test: generate
	go vet ./...
	go test ./...

build:
	go build

.PHONY: all generate test build