run:
	go run ./cmd/client .

BINARY_NAME=marketmonkey
VERSION?=0.0.1
BUILD_DIR=releases
MAIN_PATH=./cmd/client

# Build info
PACKAGE=marketmonkeyterminal
BUILD_DATE=$(shell date +%Y-%m-%d-%H:%M:%S)
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "undefined")

# Go build flags
LDFLAGS=-ldflags "-X ${PACKAGE}/version.Version=${VERSION} -X ${PACKAGE}/version.BuildDate=${BUILD_DATE} -X ${PACKAGE}/version.GitCommit=${GIT_COMMIT}"

.PHONY: all clean build build-all windows linux darwin

all: build-all

clean:
	rm -rf $(BUILD_DIR)

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# Build for all platforms
build-all: windows linux darwin

# Build for Windows
windows: $(BUILD_DIR)
	GOOS=windows go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows $(MAIN_PATH)

# Build for Linux
# linux: $(BUILD_DIR)
# 	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
# 	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)

# Build for macOS
darwin: $(BUILD_DIR)
	GOOS=darwin go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-macos $(MAIN_PATH)

# Build for current platform
build: $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)