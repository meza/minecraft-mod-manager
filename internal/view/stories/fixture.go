package stories

import "github.com/meza/minecraft-mod-manager/internal/view"

func sampleModLabel(colorMode view.ColorMode) string {
	return view.RenderModLabel(colorMode, "Sodium", "AANobbMI", "modrinth")
}
