
# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test ./...
GOGET=$(GOCMD) get
BINARY_NAME=ops

all: deps test build

build: 
	$(GOBUILD) -o $(BINARY_NAME) -v

test:
	$(GOTEST) -v

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./...
	./$(BINARY_NAME)

deps:
	$(GOGET) github.com/spf13/cobra
	$(GOGET) github.com/vishvananda/netlink
	$(GOGET) github.com/jstemmer/go-junit-report
	$(GOGET) github.com/d2g/dhcp4
	$(GOGET) github.com/d2g/dhcp4client
	$(GOGET) github.com/go-errors/errors  
	$(GOGET) github.com/cheggaaa/pb
	$(GOGET) github.com/olekukonko/tablewriter
	$(GOGET) cloud.google.com/go/storage

.PHONY: all build test clean run deps
