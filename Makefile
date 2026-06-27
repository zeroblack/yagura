BINARY := yagura
PKG := ./cmd/yagura

.PHONY: build install run test lint audit fmt tidy snapshot clean

build:
	go build -o bin/$(BINARY) $(PKG)

install:
	go install $(PKG)

run:
	go run $(PKG)

test:
	go test -race ./...

lint:
	test -z "$$(gofmt -l .)"
	go vet ./...
	go tool staticcheck ./...

audit:
	go tool govulncheck ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -rf bin dist
