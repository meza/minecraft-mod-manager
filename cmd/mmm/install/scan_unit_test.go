package install

import (
	"io"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

func TestHandlePreflightScanFailureWithPlatformLookupFailure(t *testing.T) {
	input := preflightInputs{
		deps: installDeps{
			logger: logger.New(io.Discard, io.Discard, false, false),
		},
		colorize: true,
	}
	failure := &platformLookupFailure{
		Platform:     models.MODRINTH,
		Files:        []string{"alpha.jar"},
		Reason:       "boom",
		DebugDetails: "details",
	}

	outcome, handled, err := handlePreflightScanFailure(input, failure)
	assert.NoError(t, err)
	assert.True(t, handled)
	assert.True(t, outcome.unresolved)
	assert.NotEmpty(t, outcome.lines)
}

func TestHandlePreflightScanFailurePassesThroughError(t *testing.T) {
	outcome, handled, err := handlePreflightScanFailure(preflightInputs{}, io.ErrUnexpectedEOF)
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.False(t, handled)
	assert.False(t, outcome.unresolved)
}

func TestHandlePreflightScanFailureReturnsLoggerError(t *testing.T) {
	failure := &platformLookupFailure{
		Platform:     models.MODRINTH,
		Files:        []string{"alpha.jar"},
		Reason:       "boom",
		DebugDetails: "details",
	}

	outcome, handled, err := handlePreflightScanFailure(preflightInputs{}, failure)
	assert.ErrorContains(t, err, "missing logger")
	assert.False(t, handled)
	assert.False(t, outcome.unresolved)
}

func TestReportScanResultsMarksUnmanaged(t *testing.T) {
	input := scanReportInputs{
		scanned: []scannedFile{
			{
				Path: "/mods/alpha.jar",
				Hits: []scanHit{{Platform: models.MODRINTH, Project: "alpha", Name: "Alpha"}},
			},
		},
		cfg: models.ModsJSON{Mods: []models.Mod{}},
	}

	outcome := reportScanResults(input)
	assert.True(t, outcome.unmanagedFound)
}

func TestReportScanResultsIgnoresNoHits(t *testing.T) {
	input := scanReportInputs{
		scanned: []scannedFile{{Path: "/mods/alpha.jar"}},
		cfg:     models.ModsJSON{Mods: []models.Mod{}},
	}

	outcome := reportScanResults(input)
	assert.False(t, outcome.unmanagedFound)
	assert.False(t, outcome.unresolved)
}

func TestReportScanResultsHandlesEmptyInput(t *testing.T) {
	outcome := reportScanResults(scanReportInputs{})
	assert.False(t, outcome.unmanagedFound)
	assert.False(t, outcome.unresolved)
}

func TestReportScanResultsAggregatesMultipleItems(t *testing.T) {
	input := scanReportInputs{
		colorize: true,
		scanned: []scannedFile{
			{
				Path: "/mods/unmanaged.jar",
				Sha1: "hash",
				Hits: []scanHit{{Platform: models.MODRINTH, Project: "alpha", Name: "Alpha"}},
			},
			{
				Path: "/mods/mismatch.jar",
				Sha1: "wrong-hash",
				Hits: []scanHit{{Platform: models.MODRINTH, Project: "beta", Name: "Beta"}},
			},
		},
		cfg: models.ModsJSON{Mods: []models.Mod{
			{ID: "beta", Name: "Beta", Type: models.MODRINTH},
		}},
		lock: []models.ModInstall{{
			Type: models.MODRINTH,
			ID:   "beta",
			Hash: "expected",
		}},
	}

	outcome := reportScanResults(input)
	assert.True(t, outcome.unmanagedFound)
	assert.True(t, outcome.unresolved)
	assert.Len(t, outcome.lines, 2)
}

func TestReportScanResultsMarksLockMissing(t *testing.T) {
	input := scanReportInputs{
		scanned: []scannedFile{
			{
				Path: "/mods/alpha.jar",
				Sha1: "hash",
				Hits: []scanHit{{Platform: models.MODRINTH, Project: "alpha", Name: "Alpha"}},
			},
		},
		cfg:  models.ModsJSON{Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}}},
		lock: []models.ModInstall{},
	}

	outcome := reportScanResults(input)
	assert.True(t, outcome.unresolved)
}

func TestReportScanResultsMarksHashMismatch(t *testing.T) {
	input := scanReportInputs{
		scanned: []scannedFile{
			{
				Path: "/mods/alpha.jar",
				Sha1: "wrong-hash",
				Hits: []scanHit{{Platform: models.MODRINTH, Project: "alpha", Name: "Alpha"}},
			},
		},
		cfg: models.ModsJSON{Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}}},
		lock: []models.ModInstall{{
			Type: models.MODRINTH,
			ID:   "alpha",
			Hash: "expected",
		}},
		colorize: true,
	}

	outcome := reportScanResults(input)
	assert.True(t, outcome.unresolved)
}

func TestReportScanResultsIgnoresMatchingFiles(t *testing.T) {
	input := scanReportInputs{
		scanned: []scannedFile{
			{
				Path: "/mods/alpha.jar",
				Sha1: "match",
				Hits: []scanHit{{Platform: models.MODRINTH, Project: "alpha", Name: "Alpha"}},
			},
		},
		cfg: models.ModsJSON{Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}}},
		lock: []models.ModInstall{{
			Type: models.MODRINTH,
			ID:   "alpha",
			Hash: "match",
		}},
		deps: installDeps{fs: afero.NewMemMapFs()},
	}

	outcome := reportScanResults(input)
	assert.False(t, outcome.unresolved)
	assert.False(t, outcome.unmanagedFound)
}
