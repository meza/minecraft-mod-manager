# Makefile

# Application name
APP_NAME := minecraft-mod-manager
EXECUTABLE_NAME := mmm
BUILD_DIR := build
GO_BUILD := go build -o
CROSS_CGO_ENABLED ?= 0

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

# Cross-platform helper for creating directories.
ifeq ($(OSFAMILY), Unix)
define MKDIR_P
	mkdir -p "$(1)"
endef
else
define MKDIR_P
	powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path '$(1)' | Out-Null"
endef
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

# Create build subdirectories
build-dirs: BUILD_DIR
	$(call MKDIR_P,$(BUILD_DIR)/darwin)
	$(call MKDIR_P,$(BUILD_DIR)/linux)
	$(call MKDIR_P,$(BUILD_DIR)/windows)

# Build for all platforms
build: build-dirs build-darwin build-linux build-windows

# Build for macOS
build-darwin: export GOOS := darwin
build-darwin: export GOARCH := amd64
build-darwin: export CGO_ENABLED := $(CROSS_CGO_ENABLED)
build-darwin: build-dirs
	$(GO_BUILD) $(BUILD_DIR)/darwin/$(EXECUTABLE_NAME) main.go

# Build for Linux
build-linux: export GOOS := linux
build-linux: export GOARCH := amd64
build-linux: export CGO_ENABLED := $(CROSS_CGO_ENABLED)
build-linux: build-dirs
	$(GO_BUILD) $(BUILD_DIR)/linux/$(EXECUTABLE_NAME) main.go

# Build for Windows
build-windows: export GOOS := windows
build-windows: export GOARCH := amd64
build-windows: export CGO_ENABLED := $(CROSS_CGO_ENABLED)
build-windows: build-dirs
	$(GO_BUILD) $(BUILD_DIR)/windows/$(EXECUTABLE_NAME).exe main.go

test:
	go test ./internal/...

coverage:
	go test ./internal/... -coverprofile="coverage.out"

coverage-enforce: coverage
	go tool cover -func="coverage.out" | go run tools/coverage_enforce.go

coverage-html: coverage
	go tool cover -html="coverage.out" -o coverage.html
