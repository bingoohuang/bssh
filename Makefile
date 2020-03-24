.PHONY: default test
all: default test

proxy:
	export GOPROXY=https://goproxy.cn

default: proxy
	go fmt ./...&&revive .&&goimports -w .&&golangci-lint run --enable-all&&go install -ldflags="-s -w" ./...

install: proxy
	go install -ldflags="-s -w" ./...

test: proxy
	go test ./...
