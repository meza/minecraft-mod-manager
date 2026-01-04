package add

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestRecoveryFlowSnapshots(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	t.Run("not_found_confirm", func(t *testing.T) {
		model := newRecoveryFlowModel(recoveryFlowInput{
			reason:      recoveryReasonNotFound,
			platform:    models.MODRINTH,
			projectID:   "abc",
			colorMode:   view.ColorDisabled,
			loader:      "fabric",
			gameVersion: "1.20.1",
		})

		snaps.MatchSnapshot(t, model.View())
	})

	t.Run("download_failed_confirm", func(t *testing.T) {
		model := newRecoveryFlowModel(recoveryFlowInput{
			reason:      recoveryReasonDownloadFailed,
			platform:    models.CURSEFORGE,
			projectID:   "xyz",
			colorMode:   view.ColorDisabled,
			loader:      "fabric",
			gameVersion: "1.20.1",
			retryCount:  3,
		})

		snaps.MatchSnapshot(t, model.View())
	})

	t.Run("platform_list", func(t *testing.T) {
		model := newRecoveryFlowModel(recoveryFlowInput{
			reason:    recoveryReasonNotFound,
			platform:  models.MODRINTH,
			projectID: "abc",
			colorMode: view.ColorDisabled,
		})
		model.state = recoveryStateSelectPlatform
		model.platformList.SetSize(60, 10)

		snaps.MatchSnapshot(t, model.View())
	})

	t.Run("project_id_prompt", func(t *testing.T) {
		model := newRecoveryFlowModel(recoveryFlowInput{
			reason:    recoveryReasonNotFound,
			platform:  models.MODRINTH,
			projectID: "abc",
			colorMode: view.ColorDisabled,
		})
		model.state = recoveryStateEnterProjectID
		model.selectedPlatform = models.MODRINTH

		snaps.MatchSnapshot(t, model.View())
	})
}

func TestRecoveryFlowResultFromModel(t *testing.T) {
	t.Run("aborted", func(t *testing.T) {
		result, err := recoveryFlowResultFromModel(recoveryFlowModel{
			aborted: true,
		})
		assert.Error(t, err)
		assert.Empty(t, result.projectID)
	})

	t.Run("declined", func(t *testing.T) {
		result, err := recoveryFlowResultFromModel(recoveryFlowModel{
			declined: true,
		})
		assert.Error(t, err)
		assert.Empty(t, result.projectID)
	})

	t.Run("missing_project", func(t *testing.T) {
		_, err := recoveryFlowResultFromModel(recoveryFlowModel{
			selectedPlatform: models.MODRINTH,
			selectedProject:  "",
		})
		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		result, err := recoveryFlowResultFromModel(recoveryFlowModel{
			selectedPlatform: models.MODRINTH,
			selectedProject:  "abc",
		})
		assert.NoError(t, err)
		assert.Equal(t, models.MODRINTH, result.platform)
		assert.Equal(t, "abc", result.projectID)
	})

	t.Run("unexpected_model", func(t *testing.T) {
		_, err := recoveryFlowResultFromModel(fakePromptModel{})
		assert.Error(t, err)
	})
}

func TestRunRecoveryFlowMissingRunner(t *testing.T) {
	_, err := runRecoveryFlow(recoveryFlowInput{reason: recoveryReasonNotFound})
	assert.Error(t, err)
}

func TestRunRecoveryFlowUnexpectedModel(t *testing.T) {
	runErr := errors.New("unexpected")
	_, err := runRecoveryFlow(recoveryFlowInput{
		reason: recoveryReasonNotFound,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return fakePromptModel{}, runErr
		},
	})
	assert.ErrorIs(t, err, runErr)
}

func TestRunRecoveryFlowUnexpectedResultModel(t *testing.T) {
	_, err := runRecoveryFlow(recoveryFlowInput{
		reason: recoveryReasonNotFound,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return fakePromptModel{}, nil
		},
	})
	assert.Error(t, err)
}
