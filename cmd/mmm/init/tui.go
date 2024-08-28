package init

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/term"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/tui"
	"os"
)

type options struct {
	loader       models.Loader
	gameVersion  string
	releaseTypes []models.ReleaseType
	modsFolder   string
}

type Model struct {
	form    *huh.Form
	options options
}

func (m Model) Init() tea.Cmd {
	return m.form.Init()
}

func (m Model) View() string {
	if m.form.State == huh.StateCompleted {
		return fmt.Sprintf(
			"You selected: %s with Minecraft version %s and the allowedRelease types %s",
			m.selectedLoader(),
			m.selectedGameVersion(),
			m.selectedReleaseTypes(),
		)
	}
	return m.form.View()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" {
			return m, tea.Quit
		}
	}
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	return m, cmd
}

func NewModel(loader string, gameVersion string, releaseTypes string, modsFolder string) *Model {
	loaderVal := models.Loader(loader)
	_, height, _ := term.GetSize(os.Stdout.Fd())
	loaderGroup := huh.NewGroup(loaderInput()).WithHide(isValidLoader(loaderVal))
	gameVersionGroup := huh.NewGroup(gameVersionInput()).WithHide(isValidGameVersion(gameVersion))
	releaseTypesGroup := huh.NewGroup(releaseTypesInput()).WithHide(isValidReleaseTypes(releaseTypes))
	modsFolderGroup := huh.NewGroup(getModsFolderInput()).WithHide(isValidModsFolder(modsFolder)).WithHeight(height)

	return &Model{
		options: options{
			loader:       loaderVal,
			gameVersion:  gameVersion,
			releaseTypes: parseReleaseTypes(releaseTypes),
			modsFolder:   modsFolder,
		},
		form: huh.NewForm(loaderGroup, gameVersionGroup, releaseTypesGroup, modsFolderGroup).WithKeyMap(tui.KeyMap()),
	}
}
