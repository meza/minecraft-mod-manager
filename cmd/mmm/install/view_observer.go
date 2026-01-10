package install

import "context"

type installViewObserverKey struct{}

// InstallViewObserver receives the final interactive install view after the program exits.
type InstallViewObserver func(view string)

// WithInstallViewObserver stores an observer for the interactive install view on the context.
func WithInstallViewObserver(ctx context.Context, observer InstallViewObserver) context.Context {
	if observer == nil {
		return ctx
	}
	return context.WithValue(ctx, installViewObserverKey{}, observer)
}

// NotifyInstallViewObserver calls the observer stored on the context, if any.
func NotifyInstallViewObserver(ctx context.Context, view string) {
	observer := installViewObserverFromContext(ctx)
	if observer == nil {
		return
	}
	observer(view)
}

func installViewObserverFromContext(ctx context.Context) InstallViewObserver {
	observer, ok := ctx.Value(installViewObserverKey{}).(InstallViewObserver)
	if !ok {
		return nil
	}
	return observer
}
