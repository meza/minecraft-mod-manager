package init

import "errors"

// ErrInitCanceled signals that the user intentionally exited the init flow.
var ErrInitCanceled = errors.New("init canceled")
