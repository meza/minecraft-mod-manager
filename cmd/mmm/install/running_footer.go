package install

import (
	"context"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

// RunningFooterInput provides the data needed to render a running footer line.
type RunningFooterInput struct {
	SpinnerFrame string
	ColorMode    view.ColorMode
}

// RunningFooter renders a footer line while install is running.
type RunningFooter struct {
	Render func(RunningFooterInput) string
}

type runningFooterKey struct{}

// WithRunningFooter attaches a running footer to the context for interactive install views.
func WithRunningFooter(ctx context.Context, footer RunningFooter) context.Context {
	if footer.Render == nil {
		return ctx
	}
	return context.WithValue(ctx, runningFooterKey{}, footer)
}

func runningFooterFromContext(ctx context.Context) *RunningFooter {
	footer, ok := ctx.Value(runningFooterKey{}).(RunningFooter)
	if !ok || footer.Render == nil {
		return nil
	}
	return &footer
}
