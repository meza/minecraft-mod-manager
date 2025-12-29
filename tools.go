//go:build tools
// +build tools

package tools

import (
	_ "github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod"
	_ "github.com/evilmartians/lefthook/v2"
	_ "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
	_ "github.com/google/go-licenses"
	_ "golang.org/x/vuln/cmd/govulncheck"
)
