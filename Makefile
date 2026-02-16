BINARY_NAME := mf
BUILD_DIR := bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: all build test lint clean tidy fmt

all: tidy fmt build test

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/mf

test:
	go test ./... -v -count=1

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

clean:
	rm -rf $(BUILD_DIR)

docker-build:
	docker build -t microfoundry/mf:$(VERSION) .

docker-run:
	docker run --rm -it microfoundry/mf:$(VERSION)

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)

monitoring-install:
	bash deploy/monitoring/install.sh
