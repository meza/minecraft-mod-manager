package add

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
)

func TestErrorMessageForNoFile(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	message := errorMessageForNoFile("abc", models.MODRINTH)
	assert.Contains(t, message, "cmd.add.error.no_file")
}
