GOLANG_CI_LINT_VER	:= 	v1.54.2
VERSION				:=	v0.1.0-dev

.PHONY:
build:
	go build -ldflags "-X 'github.com/meroxa/conduit-connector-amazon-sqs.version=${VERSION}'" -o conduit-connector-amazon-sqs cmd/sqs/main.go

.PHONY:
test:
	go test $(GOTEST_FLAGS) -race ./...

.PHONY: golangci-lint-install
golangci-lint-install:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANG_CI_LINT_VER)

.PHONY:
lint:
	golangci-lint run -c .golangci.yml
