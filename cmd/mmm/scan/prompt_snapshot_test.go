package scan

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/muesli/termenv"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestScanAdoptionPromptShortTerminalSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	fileNames := []string{
		"voicechat-fabric-1.21.10-2.6.11.jar",
		"viewdistancefix-fabric-1.21.10-1.0.2.jar",
		"vcinteraction-fabric-1.21.10-1.0.8.jar",
		"vanish-1.6.5+1.21.10.jar",
		"syncmatica-fabric-1.21.10-0.3.16.jar",
		"status-fabric-1.21.10-1.1.0.jar",
		"spark-1.10.152-fabric.jar",
		"servux-fabric-1.21.10-0.8.5.jar",
		"restart-detector-1.2.7+1.21.10.jar",
		"malilib-fabric-1.21.10-0.26.8.jar",
		"lithium-fabric-0.20.1+mc1.21.10.jar",
		"krypton-0.2.10.jar",
		"inventorysorter-fabric-2.1.4+mc1.21.9.jar",
		"ferritecore-8.0.2-fabric.jar",
		"fabric-language-kotlin-1.13.8+kotlin.2.3.0.jar",
		"fabric-carpet-1.21.10-1.4.188+v251016.jar",
		"fabric-api-0.138.4+1.21.10.jar",
		"entityculling-fabric-1.9.5-mc1.21.10.jar",
		"do_a_barrel_roll-fabric-3.8.3+1.21.9.jar",
		"coordfinder-fabric-1.21.10-1.1.0.jar",
		"cloth-config-20.0.149-fabric.jar",
		"cicada-lib-0.14.3+1.21.9-1.21.10.jar",
		"carpet-tis-addition-v1.74.0-mc1.21.10.jar",
		"audioplayer-fabric-2.1.0+1.21.10.jar",
		"SimpleDiscordLink-Universal-3.3.4.jar",
		"MaintenanceMode-Universal-1.3.1.jar",
		"CraterLib-Fabric-1.21.9-3.0.1.jar",
		"BetterServerPacksFabric-1.2.0.jar",
		"Axiom-5.2.1-for-MC1.21.10.jar",
		"ArmorPoser-fabric-1.21.10-12.3.0.jar",
	}

	matches := make([]scanMatch, 0, len(fileNames))
	for _, fileName := range fileNames {
		base := strings.TrimSuffix(fileName, ".jar")
		matches = append(matches, scanMatch{
			FileName:  fileName,
			Name:      base,
			ProjectID: strings.ToLower(base),
			Platform:  models.MODRINTH,
		})
	}

	sections := buildScanResultSections(scanResultsViewInput{
		matches:   matches,
		colorMode: view.ColorDisabled,
	})
	question := i18n.T("cmd.scan.prompt.add", nil)
	model := newScanAdoptionPromptModel(sections, question)

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 6})
	viewText := updated.(*scanAdoptionPromptModel).View()
	snaps.MatchSnapshot(t, viewText)
}
