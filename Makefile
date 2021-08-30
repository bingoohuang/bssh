.PHONY: default install test
all: default install test

APPNAME=bssh
VERSION=v1.0.0

app := $(notdir $(shell pwd))
goVersion := $(shell go version)
# echo ${goVersion#go version }
# strip prefix "go version " from output "go version go1.16.7 darwin/amd64"
goVersion2 := $(subst go version ,,$(goVersion))
buildTime := $(shell date '+%Y-%m-%d %H:%M:%S')
gitCommit := $(shell git rev-list -1 HEAD)
// https://stackoverflow.com/a/47510909
pkg := github.com/bingoohuang/bssh

# https://ms2008.github.io/2018/10/08/golang-build-version/
flags = "-extldflags=-s -w -X '$(pkg).buildTime=$(buildTime)' -X $(pkg).gitCommit=$(gitCommit) -X '$(pkg).goVersion=$(goVersion2)'"


gosec:
	go get github.com/securego/gosec/cmd/gosec

sec:
	@gosec ./...
	@echo "[OK] Go security check was completed!"

proxy:
	export GOPROXY=https://goproxy.cn


default: proxy
	go fmt ./...&&revive .&&goimports -w .&&golangci-lint run --enable-all

install: proxy
	go install --tags "fts5" -trimpath -ldflags=${flags} ./...
	ls -lh ~/go/bin/$(APPNAME)
	upx ~/go/bin/$(APPNAME)
	ls -lh ~/go/bin/$(APPNAME)
package: install
	mv ~/go/bin/$(APPNAME) ~/go/bin/$(APPNAME)-$(VERSION)-darwin-amd64
	gzip -f ~/go/bin/$(APPNAME)-$(VERSION)-darwin-amd64
	ls -lh ~/go/bin/$(APPNAME)*

# https://hub.docker.com/_/golang
# docker run --rm -v "$PWD":/usr/src/myapp -v "$HOME/dockergo":/go -w /usr/src/myapp golang make docker
# docker run --rm -it -v "$PWD":/usr/src/myapp -w /usr/src/myapp golang bash
# 静态连接 glibc
docker:
	docker run --rm -v "$$PWD":/usr/src/myapp -v "$$HOME/dockergo":/go -w /usr/src/myapp golang make dockerinstall
	ls -lh ~/dockergo/bin/$(APPNAME)
	upx ~/dockergo/bin/$(APPNAME)
	ls -lh ~/dockergo/bin/$(APPNAME)
	mv ~/dockergo/bin/$(APPNAME)  ~/dockergo/bin/$(APPNAME)-$(VERSION)-amd64-glibc2.28
	gzip -f ~/dockergo/bin/$(APPNAME)-$(VERSION)-amd64-glibc2.28
	ls -lh ~/dockergo/bin/$(APPNAME)*

dockerinstall: proxy
	go install -v -x -a -ldflags '-extldflags "-static" -s -w' ./...

test: proxy
	go test ./...
