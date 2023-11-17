VERSION=$(shell git describe --tags --dirty --always)
.PHONY:
build:
	go build -ldflags "-X 'github.com/conduitio-labs/conduit-connector-sqs.version=${VERSION}'" -o conduit-connector-amazon-sqs cmd/connector/main.go

.PHONY:
test:
	go test $(GOTEST_FLAGS) -race -v ./...

.PHONY: golangci-lint-install
golangci-lint-install:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY:
lint:
	golangci-lint run
	
install-paramgen:
	go install github.com/conduitio/conduit-connector-sdk/cmd/paramgen@latest
