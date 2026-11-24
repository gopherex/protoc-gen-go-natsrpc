.PHONY: build gen test integration lint bench

build:
	go build -o bin/protoc-gen-go-natsrpc .

gen: build
	PATH="$(CURDIR)/bin:$$PATH" easyp generate --config easyp.gen.yaml

test:
	go test -race ./...

integration:
	go test -race -tags integration -count=1 ./tests

lint:
	golangci-lint run --build-tags integration

bench:
	go test -run '^$$' -bench . -benchmem ./...
