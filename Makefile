VERSION=$(shell git describe --tags --dirty --always)
.PHONY:
build:
	go build -ldflags "-X 'github.com/meroxa/conduit-connector-amazon-sqs.version=${VERSION}'" -o conduit-connector-amazon-sqs cmd/sqs/main.go

.PHONY:
test:
	go test $(GOTEST_FLAGS) -race ./...

.PHONY: golangci-lint-install
golangci-lint-install:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY:
lint:
	golangci-lint run -c .golangci.yml

install-paramgen:
	go install github.com/conduitio/conduit-connector-sdk/cmd/paramgen@latest