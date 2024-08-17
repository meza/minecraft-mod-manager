# Makefile

# Application name
APP_NAME := minecraft-mod-manager
EXECUTABLE_NAME := mmm

# Build output directory
BUILD_DIR := build

# Go build command
GO_BUILD := go build -o

# Targets
.PHONY: all clean build build-darwin build-linux build-windows

# Build for all platforms
all: clean build

run:
	go run cmd/$(APP_NAME)/main.go

# Clean build directory
clean:
	go clean -cache -modcache -i -r

ifeq ($(PLATFORM), Unix)
	@if [ -d "$(BUILD_DIR)" ]; then rm -rf $(BUILD_DIR); fi
else
	@if exist $(BUILD_DIR) rmdir /S /Q $(BUILD_DIR)
endif


# Create build directory
ifeq ($(PLATFORM), Unix)

BUILD_DIR:
	@if [ ! -d "$(BUILD_DIR)" ]; then mkdir -p $(BUILD_DIR); fi

else

BUILD_DIR:
	@if not exist $(BUILD_DIR) mkdir $(BUILD_DIR)

endif

# Build for all platforms
build: BUILD_DIR build-darwin build-linux build-windows

# Build for macOS
build-darwin: BUILD_DIR
	@set GOOS=darwin
	@set GOARCH=amd64
	$(GO_BUILD) $(BUILD_DIR)/darwin/$(EXECUTABLE_NAME) cmd/$(APP_NAME)/main.go

# Build for Linux
build-linux: BUILD_DIR
	@set GOOS=linux
	@set GOARCH=amd64
	$(GO_BUILD) $(BUILD_DIR)/linux/$(EXECUTABLE_NAME) cmd/$(APP_NAME)/main.go

# Build for Windows
build-windows: BUILD_DIR
	@set GOOS=windows
	@set GOARCH=amd64
	$(GO_BUILD) $(BUILD_DIR)/windows/$(EXECUTABLE_NAME).exe cmd/$(APP_NAME)/main.go
