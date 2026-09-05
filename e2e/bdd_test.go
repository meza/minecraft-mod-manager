//go:build e2e

package e2e

import (
	"path/filepath"
	"testing"

	"github.com/cucumber/godog"
)

func TestBDDScenarios(t *testing.T) {
	options := godog.Options{
		Format: "pretty",
		Paths:  []string{filepath.Join("features")},
	}

	suite := godog.TestSuite{
		Name:                "e2e",
		ScenarioInitializer: InitializeScenario,
		Options:             &options,
	}

	status := suite.Run()
	if status != 0 {
		t.Fatalf("bdd suite failed with status %d", status)
	}
}
