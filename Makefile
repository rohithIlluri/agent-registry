.PHONY: build test vet lint clean install release-dry

BINARY := agr
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0-dev")
LDFLAGS := -s -w -X github.com/rohithilluri/agent-registry/cmd.version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

test:
	go test -race ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)
	rm -rf dist/

install: build
	install -m 0755 $(BINARY) /usr/local/bin/$(BINARY)

validate-registry:
	python3 .github/workflows/validate_registry.py

release-dry:
	goreleaser release --snapshot --clean

# Quick smoke-test against the bundled local index
smoke: build
	./$(BINARY) version
	./$(BINARY) search --index registry/index.json
	./$(BINARY) search --index registry/index.json --type mcp-server
	./$(BINARY) search --index registry/index.json git
	./$(BINARY) info io.github.community/code-review --index registry/index.json
