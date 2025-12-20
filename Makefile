# Makefile

# Application name
APP_NAME := minecraft-mod-manager
EXECUTABLE_NAME := mmm
BUILD_DIR := build
GO_BUILD := go build
CROSS_CGO_ENABLED ?= 0
BUILD_REQUIRE_TOKENS ?= 1
COVER_PACKAGES = $(filter-out github.com/meza/minecraft-mod-manager/tools,$(shell go list ./...))

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
.PHONY: all clean fmt build build-dev build-darwin build-linux build-windows

ifeq ($(OSFAMILY),Windows)
define GO_BUILD_WITH_EMBEDDED_TOKENS
	@powershell -NoProfile -NonInteractive -Command "$$ErrorActionPreference='Stop'; \
		$$repoEnvPath = Join-Path (Get-Location) '.env'; \
		$$repoEnv = @{}; \
		if (Test-Path $$repoEnvPath) { \
			Get-Content $$repoEnvPath | ForEach-Object { \
				$$line = $$_; \
				if ($$null -eq $$line) { return }; \
				$$line = $$line.Trim(); \
				if ($$line.Length -eq 0) { return }; \
				if ($$line.StartsWith('#')) { return }; \
				$$idx = $$line.IndexOf('='); \
				if ($$idx -lt 1) { return }; \
				$$key = $$line.Substring(0, $$idx).Trim(); \
				$$val = $$line.Substring($$idx + 1).Trim(); \
				if ($$val.Length -ge 2) { \
					$$first = $$val[0]; \
					$$last = $$val[$$val.Length - 1]; \
					if (($$first -eq [char]34 -and $$last -eq [char]34) -or ($$first -eq [char]39 -and $$last -eq [char]39)) { \
						$$val = $$val.Substring(1, $$val.Length - 2); \
					}; \
				}; \
				$$repoEnv[$$key] = $$val; \
			}; \
		}; \
		$$tokenNames = @('MODRINTH_API_KEY','CURSEFORGE_API_KEY','POSTHOG_API_KEY'); \
		foreach ($$name in $$tokenNames) { \
			if (Test-Path (\"Env:\" + $$name)) { continue }; \
			if ($$repoEnv.ContainsKey($$name)) { Set-Item -Path (\"Env:\" + $$name) -Value $$repoEnv[$$name] }; \
		}; \
		$$missing = @(); \
		foreach ($$name in $$tokenNames) { \
			$$current = (Get-Item -Path (\"Env:\" + $$name) -ErrorAction SilentlyContinue).Value; \
			if ([string]::IsNullOrEmpty($$current)) { $$missing += $$name }; \
		}; \
		if (($(BUILD_REQUIRE_TOKENS)) -eq 1 -and $$missing.Count -gt 0) { \
			throw (\"error: missing build token(s): \" + ($$missing -join ' ') + \". hint: set them as environment variables or add them to ./.env (repo root) before running make\"); \
		}; \
		if (($(BUILD_REQUIRE_TOKENS)) -ne 1 -and $$missing.Count -gt 0) { \
			Write-Warning (\"warning: building without embedded token(s): \" + ($$missing -join ' ') + \" (runtime overrides required)\"); \
		}; \
		$$ldflagsParts = @(); \
		if ($$env:MODRINTH_API_KEY) { $$ldflagsParts += ('-X github.com/meza/minecraft-mod-manager/internal/environment.modrinthApiKeyDefault=' + $$env:MODRINTH_API_KEY) }; \
		if ($$env:CURSEFORGE_API_KEY) { $$ldflagsParts += ('-X github.com/meza/minecraft-mod-manager/internal/environment.curseforgeApiKeyDefault=' + $$env:CURSEFORGE_API_KEY) }; \
		if ($$env:POSTHOG_API_KEY) { $$ldflagsParts += ('-X github.com/meza/minecraft-mod-manager/internal/environment.posthogApiKeyDefault=' + $$env:POSTHOG_API_KEY) }; \
		$$ldflags = ($$ldflagsParts -join ' '); \
		$(GO_BUILD) -ldflags \"$$ldflags\" -o $(1) main.go"
endef
else
define GO_BUILD_WITH_EMBEDDED_TOKENS
	@set -eu; \
	modrinth_was_set=0; if [ "$${MODRINTH_API_KEY+x}" = x ]; then modrinth_was_set=1; modrinth_value="$$MODRINTH_API_KEY"; fi; \
	curseforge_was_set=0; if [ "$${CURSEFORGE_API_KEY+x}" = x ]; then curseforge_was_set=1; curseforge_value="$$CURSEFORGE_API_KEY"; fi; \
	posthog_was_set=0; if [ "$${POSTHOG_API_KEY+x}" = x ]; then posthog_was_set=1; posthog_value="$$POSTHOG_API_KEY"; fi; \
	if [ -f .env ]; then set -a; . ./.env; set +a; fi; \
	if [ "$$modrinth_was_set" -eq 1 ]; then MODRINTH_API_KEY="$$modrinth_value"; fi; \
	if [ "$$curseforge_was_set" -eq 1 ]; then CURSEFORGE_API_KEY="$$curseforge_value"; fi; \
	if [ "$$posthog_was_set" -eq 1 ]; then POSTHOG_API_KEY="$$posthog_value"; fi; \
	missing=""; \
	if [ -z "$${MODRINTH_API_KEY:-}" ]; then missing="$$missing MODRINTH_API_KEY"; fi; \
	if [ -z "$${CURSEFORGE_API_KEY:-}" ]; then missing="$$missing CURSEFORGE_API_KEY"; fi; \
	if [ -z "$${POSTHOG_API_KEY:-}" ]; then missing="$$missing POSTHOG_API_KEY"; fi; \
	if [ "$(BUILD_REQUIRE_TOKENS)" = "1" ] && [ -n "$$missing" ]; then \
		echo "error: missing build token(s):$$missing" >&2; \
		echo "hint: set them as environment variables or add them to ./.env (repo root) before running make" >&2; \
		exit 1; \
	fi; \
	if [ "$(BUILD_REQUIRE_TOKENS)" != "1" ] && [ -n "$$missing" ]; then \
		echo "warning: building without embedded token(s):$$missing (runtime overrides required)" >&2; \
	fi; \
	LDFLAGS=""; \
	if [ -n "$${MODRINTH_API_KEY:-}" ]; then LDFLAGS="$$LDFLAGS -X github.com/meza/minecraft-mod-manager/internal/environment.modrinthApiKeyDefault=$${MODRINTH_API_KEY}"; fi; \
	if [ -n "$${CURSEFORGE_API_KEY:-}" ]; then LDFLAGS="$$LDFLAGS -X github.com/meza/minecraft-mod-manager/internal/environment.curseforgeApiKeyDefault=$${CURSEFORGE_API_KEY}"; fi; \
	if [ -n "$${POSTHOG_API_KEY:-}" ]; then LDFLAGS="$$LDFLAGS -X github.com/meza/minecraft-mod-manager/internal/environment.posthogApiKeyDefault=$${POSTHOG_API_KEY}"; fi; \
	$(GO_BUILD) -ldflags "$$LDFLAGS" -o $(1) main.go
endef
endif

# Build for all platforms
all: clean build

run:
	go run .

fmt:
	go fmt ./...

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

build-dev: BUILD_REQUIRE_TOKENS=0
build-dev: build

# Build for macOS
build-darwin: export GOOS := darwin
build-darwin: export GOARCH := amd64
build-darwin: export CGO_ENABLED := $(CROSS_CGO_ENABLED)
build-darwin: build-dirs
	$(call GO_BUILD_WITH_EMBEDDED_TOKENS,$(BUILD_DIR)/darwin/$(EXECUTABLE_NAME))

# Build for Linux
build-linux: export GOOS := linux
build-linux: export GOARCH := amd64
build-linux: export CGO_ENABLED := $(CROSS_CGO_ENABLED)
build-linux: build-dirs
	$(call GO_BUILD_WITH_EMBEDDED_TOKENS,$(BUILD_DIR)/linux/$(EXECUTABLE_NAME))

# Build for Windows
build-windows: export GOOS := windows
build-windows: export GOARCH := amd64
build-windows: export CGO_ENABLED := $(CROSS_CGO_ENABLED)
build-windows: build-dirs
	$(call GO_BUILD_WITH_EMBEDDED_TOKENS,$(BUILD_DIR)/windows/$(EXECUTABLE_NAME).exe)

test:
	go test ./...

coverage:
	go test $(COVER_PACKAGES) -coverprofile="coverage.out"

coverage-enforce: coverage
	go tool cover -func="coverage.out" | go run tools/coverage_enforce.go

coverage-html: coverage
	go tool cover -html="coverage.out" -o coverage.html
