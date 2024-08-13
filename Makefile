# Makefile for building norn and norn-client

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME_NORN=norn
BINARY_NAME_NORN_CLIENT=norn-client

# Directories
NORN_DIR=./cmd/cli
NORN_CLIENT_DIR=./cmd/client

# Build norn
build-norn:
	CGO_ENABLED=0 $(GOBUILD) -o $(BINARY_NAME_NORN) -v $(NORN_DIR)

# Build norn-client
build-norn-client:
	CGO_ENABLED=0 $(GOBUILD) -o $(BINARY_NAME_NORN_CLIENT) -v $(NORN_CLIENT_DIR)

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME_NORN) $(BINARY_NAME_NORN_CLIENT)

# Run tests
test:
	$(GOTEST) -v ./...

# Default target
all: build-norn build-norn-client