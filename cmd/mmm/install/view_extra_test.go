package install

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestRenderInstallRunningViewIncludesFooterLine(t *testing.T) {
	items := []installItem{
		{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, DisplayName: "Alpha", Status: installItemPending},
	}

	output := renderInstallRunningView(view.ColorDisabled, items, "footer line")

	assert.Contains(t, output, "footer line")
}
