package install

import (
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestOptionalStringValue(t *testing.T) {
	assert.Equal(t, "", optionalStringValue(nil))
	value := "  value "
	assert.Equal(t, "value", optionalStringValue(&value))
}

func TestModVersionLabel(t *testing.T) {
	version := "1.2.3"
	mod := models.Mod{Version: &version}
	assert.Equal(t, "1.2.3", modVersionLabel(mod))

	blank := "   "
	mod.Version = &blank
	assert.Equal(t, "latest", modVersionLabel(mod))
}
