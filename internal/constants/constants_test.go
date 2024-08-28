package constants

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConstants(t *testing.T) {
	assert.Equal(t, "minecraft-mod-manager", APP_NAME, "APP_NAME should be 'minecraft-mod-manager'")
	assert.Equal(t, "mmm", COMMAND_NAME, "COMMAND_NAME should be 'mmm'")
}
