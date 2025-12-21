# Makefile

# Application name
APP_NAME := minecraft-mod-manager
EXECUTABLE_NAME := mmm
BUILD_DIR := build
VERSION ?= dev
GOLANGCI_LINT_TOOLCHAIN := go1.25.0

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
.PHONY: all clean fmt lint lint-fix build dist prepare test coverage

# Build for all platforms
all: clean build

run:
	go run .

fmt:
	go fmt ./...

lint:
ifeq ($(OSFAMILY), Windows)
	@powershell -NoProfile -Command "$$env:GOTOOLCHAIN='$(GOLANGCI_LINT_TOOLCHAIN)'; go run github.com/golangci/golangci-lint/cmd/golangci-lint run"
else
	@GOTOOLCHAIN=$(GOLANGCI_LINT_TOOLCHAIN) go run github.com/golangci/golangci-lint/cmd/golangci-lint run
endif

lint-fix:
ifeq ($(OSFAMILY), Windows)
	@powershell -NoProfile -Command "$$env:GOTOOLCHAIN='$(GOLANGCI_LINT_TOOLCHAIN)'; go run github.com/golangci/golangci-lint/cmd/golangci-lint run --fix"
else
	@GOTOOLCHAIN=$(GOLANGCI_LINT_TOOLCHAIN) go run github.com/golangci/golangci-lint/cmd/golangci-lint run --fix
endif

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


build:
	go run ./tools/build

dist:
	go run ./tools/packaging --version "$(VERSION)"

prepare:
	$(MAKE) build VERSION="$(VERSION)"
	$(MAKE) dist VERSION="$(VERSION)"

test:
	go test ./...

coverage:
	go run ./tools/coverage
