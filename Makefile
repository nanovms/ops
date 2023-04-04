
# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test ./...
GOGET=$(GOCMD) get
BINARY_NAME=ops

all: deps test build

pre-build:

do-build: pre-build
	$(GOBUILD) -o $(BINARY_NAME) -v

post-build: do-build

build: post-build

pre-test:

do-test: pre-test
	@GO111MODULE=on $(GOTEST) -v

post-test: do-test

test: post-test


# go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
# go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
# go install github.com/bufbuild/buf/cmd/buf@latest
generate:
	buf generate --path ./protos/imageservice/imageservice.proto
	buf generate --path ./protos/instanceservice/instanceservice.proto
	buf generate --path ./protos/volumeservice/volumeservice.proto

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf protos/imageservice/*.go
	rm -rf protos/imageservice/*.json
	rm -rf protos/instanceservice/*.go
	rm -rf protos/instanceservice/*.json
	rm -rf protos/volumeservice/*.go
	rm -rf protos/volumeservice/*.json

run:
	$(GOBUILD) -o $(BINARY_NAME) -v .
	./$(BINARY_NAME)

deps:
	@GO111MODULE=on go mod download

# go install github.com/golang/mock/mockgen@latest
update-mocks:
	go generate network/setup_network_interfaces.go
	go generate provider/upcloud/upcloud.go
	go generate provider/oci/oci.go

.PHONY: all build test clean run deps
.PHONY: pre-build do-build post-build
.PHONY: pre-test do-test post-test

-include Makefile.local
