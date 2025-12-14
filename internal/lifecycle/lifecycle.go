package lifecycle

import (
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
)

// Handler receives the OS signal that triggered shutdown.
type Handler func(os.Signal)

// HandlerID identifies a registered handler.
type HandlerID int64

var (
	defaultSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

	handlerCounter atomic.Int64

	startOnce  sync.Once
	signalChan chan os.Signal

	handlersMu sync.RWMutex
	handlers   = make(map[HandlerID]Handler)
	order      []HandlerID

	channelFactory = newSignalChan
	notifyFunc     = signal.Notify
	stopFunc       = signal.Stop
	exitFunc       = os.Exit
)

// Register adds a handler that will run when a shutdown signal arrives.
// Handlers execute in reverse registration order. The returned HandlerID can
// be passed to Unregister to remove the handler.
func Register(handler Handler) HandlerID {
	if handler == nil {
		return 0
	}

	startOnce.Do(startListener)

	id := HandlerID(handlerCounter.Add(1))

	handlersMu.Lock()
	handlers[id] = handler
	order = append(order, id)
	handlersMu.Unlock()

	return id
}

// Unregister removes a previously registered handler.
func Unregister(id HandlerID) {
	if id == 0 {
		return
	}

	handlersMu.Lock()
	defer handlersMu.Unlock()

	delete(handlers, id)
	for i, existing := range order {
		if existing == id {
			order = append(order[:i], order[i+1:]...)
			break
		}
	}
}

func startListener() {
	signalChan = channelFactory()
	notifyFunc(signalChan, defaultSignals...)

	go func() {
		sig := <-signalChan
		runHandlers(sig)
		exitFunc(exitCode(sig))
	}()
}

func runHandlers(sig os.Signal) {
	handlersMu.RLock()
	snapshot := make([]HandlerID, len(order))
	copy(snapshot, order)
	handlerCopy := make(map[HandlerID]Handler, len(handlers))
	for id, handler := range handlers {
		handlerCopy[id] = handler
	}
	handlersMu.RUnlock()

	for i := len(snapshot) - 1; i >= 0; i-- {
		if handler := handlerCopy[snapshot[i]]; handler != nil {
			callHandler(handler, sig)
		}
	}
}

func callHandler(handler Handler, sig os.Signal) {
	defer func() {
		if recover() != nil {
			// swallow panics so remaining handlers run
		}
	}()
	handler(sig)
}

func exitCode(sig os.Signal) int {
	switch sig {
	case os.Interrupt:
		return 130
	case syscall.SIGTERM:
		return 143
	default:
		return 1
	}
}

// reset clears global state (tests only).
func reset() {
	if signalChan != nil {
		stopFunc(signalChan)
	}
	signalChan = nil

	startOnce = sync.Once{}
	handlerCounter.Store(0)

	handlersMu.Lock()
	handlers = make(map[HandlerID]Handler)
	order = nil
	handlersMu.Unlock()

	restoreFactories()
}

func newSignalChan() chan os.Signal {
	return make(chan os.Signal, 1)
}

func restoreFactories() {
	channelFactory = newSignalChan
	notifyFunc = signal.Notify
	stopFunc = signal.Stop
	exitFunc = os.Exit
}
