# Makefile

# Application name
APP_NAME := minecraft-mod-manager
EXECUTABLE_NAME := mmm
BUILD_DIR := build
GO_BUILD := go build -o

ifeq ($(OS),Windows_NT)
        OSFLAG  := WIN
        OSFAMILY := Windows
        CCFLAGS += -D WIN32
        ifeq ($(PROCESSOR_ARCHITEW6432),AMD64)
                CCFLAGS += -D AMD64
        else
                ifeq ($(PROCESSOR_ARCHITECTURE),AMD64)
                        CCFLAGS += -D AMD64
                endif
                ifeq ($(PROCESSOR_ARCHITECTURE),x86)
                        CCFLAGS += -D IA32
                endif
        endif
else
        UNAME_S := $(shell uname -s)
        OSFAMILY := Unix
        ifeq ($(UNAME_S),Linux)
                OSFLAG := Linux
                CCFLAGS += -D LINUX
        endif
        ifeq ($(UNAME_S),Darwin)
                OSFLAG := Darwin
                CCFLAGS += -D OSX
        endif
                UNAME_P := $(shell uname -p)
        ifeq ($(UNAME_P),x86_64)
                CCFLAGS += -D AMD64
        endif
        ifneq ($(filter %86,$(UNAME_P)),)
                CCFLAGS += -D IA32
        endif
endif

# Targets
.PHONY: all clean build build-darwin build-linux build-windows

# Build for all platforms
all: clean build

run:
	go run .

# Clean build directory
ifeq ($(OSFAMILY), Unix)
clean:
	go clean -cache -modcache -i -r
	if [ -d "$(BUILD_DIR)" ]; then rm -rf $(BUILD_DIR); fi
else
clean:
	go clean -cache -modcache -i -r
	@if exist $(BUILD_DIR) rmdir /S /Q $(BUILD_DIR)
endif


# Create build directory
ifeq ($(OSFAMILY), Unix)
BUILD_DIR:
	if [ ! -d "$(BUILD_DIR)" ]; then mkdir -p $(BUILD_DIR); fi
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
	$(GO_BUILD) $(BUILD_DIR)/darwin/$(EXECUTABLE_NAME) main.go

# Build for Linux
build-linux: BUILD_DIR
	@set GOOS=linux
	@set GOARCH=amd64
	$(GO_BUILD) $(BUILD_DIR)/linux/$(EXECUTABLE_NAME) main.go

# Build for Windows
build-windows: BUILD_DIR
	@set GOOS=windows
	@set GOARCH=amd64
	$(GO_BUILD) $(BUILD_DIR)/windows/$(EXECUTABLE_NAME).exe main.go

test:
	go test ./internal/...

coverage:
	go test ./internal/... -coverprofile="coverage.out"

coverage-enforce: coverage
	go tool cover -func="coverage.out" | go run tools/coverage_enforce.go

coverage-html: coverage
	go tool cover -html="coverage.out" -o coverage.html
