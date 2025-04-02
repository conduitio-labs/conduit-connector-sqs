VERSION=$(shell git describe --tags --dirty --always)

.PHONY: build
build:
	go build -ldflags "-X 'github.com/conduitio-labs/conduit-connector-sqs.version=${VERSION}'" -o conduit-connector-sqs cmd/connector/main.go

.PHONY: test
test:
	go test $(GOTEST_FLAGS) -race -v ./...

test-integration: up
	go test $(GOTEST_FLAGS) -v -race ./...; ret=$$?; \
		docker compose -f test/docker-compose.yml down -v; \
		exit $$ret

.PHONY: generate
generate:
	go generate ./...
	conn-sdk-cli readmegen -w

.PHONY: lint
lint:
	golangci-lint run

.PHONY: install-tools
install-tools:
	@echo Installing tools from tools/go.mod
	@go list -modfile=tools/go.mod tool | xargs -I % go list -modfile=tools/go.mod -f "%@{{.Module.Version}}" % | xargs -tI % go install %
	@go mod tidy

.PHONY: up
up:
	docker compose -f test/docker-compose.yml up --quiet-pull -d --wait 

.PHONY: down
down:
	docker compose -f test/docker-compose.yml down -v --remove-orphans