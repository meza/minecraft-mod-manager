package lifecycle

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHandlersRunInReverseOrder(t *testing.T) {
	withSignalHarness(t, func(sigCh chan os.Signal, exitCh chan int) {
		var calls []string
		Register(func(os.Signal) { calls = append(calls, "first") })
		Register(func(os.Signal) { calls = append(calls, "second") })

		sigCh <- syscall.SIGINT
		waitExit(t, exitCh, syscall.SIGINT)
		assert.Equal(t, []string{"second", "first"}, calls)
	})
}

func TestUnregisterPreventsInvocation(t *testing.T) {
	withSignalHarness(t, func(sigCh chan os.Signal, exitCh chan int) {
		var called bool
		id := Register(func(os.Signal) { called = true })
		Unregister(id)

		sigCh <- syscall.SIGTERM
		waitExit(t, exitCh, syscall.SIGTERM)
		assert.False(t, called)
	})
}

func TestPanicsAreSwallowed(t *testing.T) {
	withSignalHarness(t, func(sigCh chan os.Signal, exitCh chan int) {
		var called bool
		Register(func(os.Signal) { panic("boom") })
		Register(func(os.Signal) { called = true })

		sigCh <- syscall.SIGINT
		waitExit(t, exitCh, syscall.SIGINT)
		assert.True(t, called)
	})
}

func TestRegisterIgnoresNil(t *testing.T) {
	withSignalHarness(t, func(sigCh chan os.Signal, exitCh chan int) {
		assert.Equal(t, HandlerID(0), Register(nil))
		Register(func(os.Signal) {})

		sigCh <- syscall.SIGINT
		waitExit(t, exitCh, syscall.SIGINT)
	})
}

func TestExitCodeMappings(t *testing.T) {
	assert.Equal(t, 130, exitCode(os.Interrupt))
	assert.Equal(t, 143, exitCode(syscall.SIGTERM))
	assert.Equal(t, 1, exitCode(syscall.Signal(0)))
}

func TestUnregisterIgnoresZero(t *testing.T) {
	reset()
	t.Cleanup(reset)
	Unregister(0)
}

func TestResetStopsSignalListener(t *testing.T) {
	reset()
	defer reset()

	stopped := make(chan struct{}, 1)
	stopFunc = func(c chan<- os.Signal) {
		stopped <- struct{}{}
	}

	channelFactory = func() chan os.Signal { return make(chan os.Signal, 1) }
	Register(func(os.Signal) {})

	reset()

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("stopFunc was not called")
	}
}

func TestResetWhenNoListenerIsActive(t *testing.T) {
	reset()
	defer reset()

	stopCalled := false
	stopFunc = func(chan<- os.Signal) {
		stopCalled = true
	}

	reset()

	assert.False(t, stopCalled)
}

func TestNewSignalChanBuffered(t *testing.T) {
	ch := newSignalChan()
	require := assert.New(t)
	require.NotNil(ch)
	select {
	case ch <- os.Interrupt:
	default:
		t.Fatal("channel should be buffered")
	}
}

func withSignalHarness(t *testing.T, fn func(chan os.Signal, chan int)) {
	reset()
	t.Cleanup(reset)

	sigCh := make(chan os.Signal, 1)
	channelFactory = func() chan os.Signal { return sigCh }

	exitCh := make(chan int, 1)
	exitFunc = func(code int) { exitCh <- code }

	fn(sigCh, exitCh)
}

func waitExit(t *testing.T, exitCh chan int, sig os.Signal) {
	t.Helper()
	select {
	case code := <-exitCh:
		assert.Equal(t, exitCode(sig), code)
	case <-time.After(time.Second):
		t.Fatalf("signal %v not handled", sig)
	}
}
