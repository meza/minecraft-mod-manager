# Makefile

# Application name
APP_NAME := minecraft-mod-manager
EXECUTABLE_NAME := mmm
BUILD_DIR := build
E2E_BUILD_DIR := $(BUILD_DIR)/e2e
METADATA_DIR := $(BUILD_DIR)/metadata
NOTICES_FILE := $(METADATA_DIR)/THIRD_PARTY_NOTICES.txt
SBOM_FILE := $(METADATA_DIR)/mmm-sbom.json
VERSION ?= dev
TEST ?= ./...
E2E_TEST ?= ./e2e ./internal/i18n
TUI_TEST_BIN ?=
export TUI_TEST_BIN
ifeq ($(OS),Windows_NT)
        E2E_EXECUTABLE := $(E2E_BUILD_DIR)/mmm.exe
else
        E2E_EXECUTABLE := $(E2E_BUILD_DIR)/mmm
endif
GOLANGCI_LINT_TOOLCHAIN := go1.25.5
GOVULNCHECK_TOOLCHAIN := go1.25.5

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
.PHONY: all clean fmt fmt-check lint lint-fix vuln build e2e-build dist prepare test test-race coverage mod-download notices sbom vcr-record e2e

# Build for all platforms
all: clean build

run:
	go run .

fmt:
	go fmt ./...

fmt-check:
ifeq ($(OSFAMILY), Windows)
	@powershell -NoProfile -Command "$$files = gofmt -l .; if ($$files) { Write-Host 'Run gofmt on:'; $$files; exit 1 }"
else
	@files=$$(gofmt -l .); if [ -n "$$files" ]; then echo 'Run gofmt on:'; echo "$$files"; exit 1; fi
endif

lint:
ifeq ($(OSFAMILY), Windows)
	@powershell -NoProfile -Command "$$env:GOTOOLCHAIN='$(GOLANGCI_LINT_TOOLCHAIN)'; go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint run"
else
	@GOTOOLCHAIN=$(GOLANGCI_LINT_TOOLCHAIN) go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint run
endif

lint-fix:
ifeq ($(OSFAMILY), Windows)
	@powershell -NoProfile -Command "$$env:GOTOOLCHAIN='$(GOLANGCI_LINT_TOOLCHAIN)'; go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint run --fix"
else
	@GOTOOLCHAIN=$(GOLANGCI_LINT_TOOLCHAIN) go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint run --fix
endif

# NOTE: govulncheck symbol scan panics under Go 1.25.x; see https://github.com/golang/go/issues/73871
vuln:
ifeq ($(OSFAMILY), Windows)
	@powershell -NoProfile -Command "$$env:GOTOOLCHAIN='$(GOVULNCHECK_TOOLCHAIN)'; go run golang.org/x/vuln/cmd/govulncheck -scan=package ./..."
else
	@GOTOOLCHAIN=$(GOVULNCHECK_TOOLCHAIN) go run golang.org/x/vuln/cmd/govulncheck -scan=package ./...
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

e2e-build:
	$(call MKDIR_P,$(E2E_BUILD_DIR))
	go build -tags=e2e -o "$(E2E_EXECUTABLE)" .

notices:
	$(call MKDIR_P,$(METADATA_DIR))
ifeq ($(OSFAMILY), Windows)
	@powershell -NoProfile -Command "$$env:GOTOOLCHAIN='$(GOVULNCHECK_TOOLCHAIN)'; go run github.com/google/go-licenses@v1.6.0 report ./... | Out-File -FilePath '$(NOTICES_FILE)' -Encoding utf8"
else
	@GOTOOLCHAIN=$(GOVULNCHECK_TOOLCHAIN) go run github.com/google/go-licenses@v1.6.0 report ./... > "$(NOTICES_FILE)"
endif

sbom:
	$(call MKDIR_P,$(METADATA_DIR))
ifeq ($(OSFAMILY), Windows)
	@powershell -NoProfile -Command "$$env:GOTOOLCHAIN='$(GOVULNCHECK_TOOLCHAIN)'; go run github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.9.0 mod -licenses -json -output '$(SBOM_FILE)'"
else
	@GOTOOLCHAIN=$(GOVULNCHECK_TOOLCHAIN) go run github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.9.0 mod -licenses -json -output "$(SBOM_FILE)"
endif

dist: notices sbom
	go run ./tools/packaging --version "$(VERSION)"

prepare:
	$(MAKE) build VERSION="$(VERSION)"
	$(MAKE) dist VERSION="$(VERSION)"

test:
	go test ./...

vcr-record:
	MMM_RECORD_HTTP=1 go test -count=1 $(TEST)

e2e: e2e-build
	go test -tags=e2e $(E2E_TEST)

test-race:
	go test -race ./...

coverage:
	go run ./tools/coverage

mod-download:
	go mod download
