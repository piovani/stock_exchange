PROTO_DIR := proto
PB_DIR    := pb
PROTO_FILES := $(shell find $(PROTO_DIR) -name "*.proto")

.PHONY: proto build run test lint

proto:
	PATH="$$PATH:$(shell go env GOPATH)/bin" protoc \
		--go_out=$(PB_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(PB_DIR) \
		--go-grpc_opt=paths=source_relative \
		--proto_path=$(PROTO_DIR) \
		$(PROTO_FILES)

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./...

lint:
	go vet ./...
