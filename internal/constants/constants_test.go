package constants

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConstants(t *testing.T) {
	assert.Equal(t, "minecraft-mod-manager", AppName, "AppName should be 'minecraft-mod-manager'")
	assert.Equal(t, "mmm", CommandName, "CommandName should be 'mmm'")
}
