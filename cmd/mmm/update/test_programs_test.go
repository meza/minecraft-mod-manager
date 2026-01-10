package update

import (
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMain(m *testing.M) {
	originalRunUpdateProgram := runUpdateProgram
	originalRunUpdateTranscriptProgram := runUpdateTranscriptProgram

	runUpdateProgram = runUpdateProgramForTests
	runUpdateTranscriptProgram = runUpdateTranscriptProgramForTests

	exitCode := m.Run()

	runUpdateProgram = originalRunUpdateProgram
	runUpdateTranscriptProgram = originalRunUpdateTranscriptProgram

	os.Exit(exitCode)
}

func runUpdateProgramForTests(model *updateModel, _ ...tea.ProgramOption) (tea.Model, error) {
	model.bindSender(func(msg tea.Msg) {
		typed, ok := msg.(updateItemStatusMsg)
		if !ok {
			return
		}
		model.applyItemUpdate(typed)
	})

	outcome := model.execRunner(model.ctx, model.sender)
	model.items = outcome.items
	model.outcome = outcome
	model.finalRender = true
	return model, nil
}

func runUpdateTranscriptProgramForTests(model *updateTranscriptModel, _ ...tea.ProgramOption) (tea.Model, error) {
	model.bindSender(func(msg tea.Msg) {
		typed, ok := msg.(updateItemStatusMsg)
		if !ok {
			return
		}
		if line, ok := model.applyTranscriptUpdate(typed); ok {
			runTeaCmd(outputLineCmd(model.output, line))
		}
	})

	outcome := model.execRunner(model.ctx, model.sender)
	model.items = outcome.items
	model.outcome = outcome
	runTeaCmd(summaryLinesCmd(model.output, model.summaryLines()))
	return model, nil
}
