package install

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

type installProgramOptionsKey struct{}

// WithInstallProgramOptions attaches Bubble Tea program options for install runs.
func WithInstallProgramOptions(ctx context.Context, options ...tea.ProgramOption) context.Context {
	if len(options) == 0 {
		return ctx
	}
	return context.WithValue(ctx, installProgramOptionsKey{}, options)
}

func installProgramOptionsFromContext(ctx context.Context) []tea.ProgramOption {
	options, ok := ctx.Value(installProgramOptionsKey{}).([]tea.ProgramOption)
	if !ok || len(options) == 0 {
		return nil
	}
	return options
}
