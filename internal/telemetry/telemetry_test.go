package telemetry

import (
	"errors"
	"os"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/posthog/posthog-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockClient is a mock implementation of the Client interface
type MockClient struct {
	mock.Mock
}

func (m *MockClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockClient) Enqueue(msg posthog.Message) error {
	args := m.Called(msg)
	return args.Error(0)
}

func TestGetMachineId(t *testing.T) {
	t.Run("Machine ID is set", func(t *testing.T) {
		t.Setenv("MACHINE_ID", "test-machine-id")
		assert.Equal(t, "test-machine-id", getMachineId())

	})

	t.Run("Machine ID is not set", func(t *testing.T) {
		old := os.Getenv("MACHINE_ID")
		os.Unsetenv("MACHINE_ID")
		t.Cleanup(func() { os.Setenv("MACHINE_ID", old) })

		mid := getMachineId()
		assert.NotEmpty(t, mid)
		assert.NotEqual(t, "test-machine-id", mid)
	})
}

func TestCapture(t *testing.T) {
	mockClient := new(MockClient)
	mockClient.On("Enqueue", mock.Anything).Return(nil)
	mockClient.On("Close").Return(nil)
	singleClient = mockClient

	Capture("test-event", map[string]interface{}{"key": "value"})

	mockClient.AssertCalled(t, "Enqueue", mock.Anything)
	mockClient.AssertCalled(t, "Close")
}

func TestCaptureCommand(t *testing.T) {
	mockClient := new(MockClient)
	mockClient.On("Enqueue", mock.Anything).Return(nil)
	mockClient.On("Close").Return(nil)

	singleClient = mockClient

	command := CommandTelemetry{
		Command: "test-command",
		Success: false,
		Config:  &models.ModsJson{},
		Error:   errors.New("test error"),
		Extra:   map[string]interface{}{"extra": "data"},
	}

	CaptureCommand(command)

	mockClient.AssertCalled(t, "Enqueue", mock.Anything)
	mockClient.AssertCalled(t, "Close")
}

func TestInitClient(t *testing.T) {
	t.Run("Client is not re-initialized", func(t *testing.T) {
		mockClient := new(MockClient)
		mockClient.On("Close").Return(nil)
		mockClient.On("Enqueue", mock.Anything).Return(nil)

		singleClient = mockClient

		client := initClient()
		assert.NotNil(t, client)

		client2 := initClient()
		assert.Equal(t, client, client2)

		mockClient.AssertNumberOfCalls(t, "Close", 0)
		mockClient.AssertNumberOfCalls(t, "Enqueue", 0)
	})

	t.Run("Client is initialized correctly when not mocked", func(t *testing.T) {
		singleClient = nil

		client := initClient()
		assert.NotNil(t, client)
		assert.NotNil(t, machineId)
	})
}
