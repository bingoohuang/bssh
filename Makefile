# Go コマンド
GOCMD=go
MODULE=GO111MODULE=on
GOBUILD=$(MODULE) $(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(MODULE) $(GOCMD) test -cover
GOGET=$(GOCMD) get
GOMOD=$(MODULE) $(GOCMD) mod
GOINSTALL=$(MODULE) $(GOCMD) install

build:
	# Remove unnecessary dependent libraries
	$(GOMOD) tidy
	# Place dependent libraries under vendor
	$(GOMOD) vendor
	# Build bssh
	$(GOBUILD) ./cmd/bssh
	# Build lscp
	$(GOBUILD) ./cmd/lscp
	# Build lsftp
	$(GOBUILD) ./cmd/lsftp

clean:
	$(GOCLEAN) ./...
	rm -f bssh
	rm -f lscp
	rm -f lsftp

install:
	# copy bssh binary to /usr/local/bin/
	cp bssh /usr/local/bin/
	# copy lscp binary to /usr/local/bin/
	cp lscp /usr/local/bin/
	# copy lsftp binary to /usr/local/bin/
	cp lsftp /usr/local/bin/
	cp -n example/config.tml ~/.bssh.conf || true

test:
	$(GOTEST) ./...
