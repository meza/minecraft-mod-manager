package init

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/tui"
)

type ReleaseTypesSelectedMessage struct {
	ReleaseTypes []models.ReleaseType
}

type ReleaseTypesModel struct {
	list     list.Model
	selected map[models.ReleaseType]bool
	error    error
	Value    []models.ReleaseType
}

func NewReleaseTypesModel(defaults []models.ReleaseType) ReleaseTypesModel {
	width, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		width = 0
	}

	selected := make(map[models.ReleaseType]bool, len(defaults))
	for _, value := range defaults {
		selected[value] = true
	}

	allReleaseTypes := models.AllReleaseTypes()
	items := make([]list.Item, 0, len(allReleaseTypes))
	for _, releaseType := range allReleaseTypes {
		items = append(items, releaseTypeItem{
			value:    releaseType,
			selected: selected,
		})
	}

	listModel := list.New(items, releaseTypeDelegate{}, width, 14)
	listModel.Title = tui.QuestionStyle.Render("? ") + tui.TitleStyle.Render(i18n.T("cmd.init.tui.release-types.question"))
	listModel.SetShowStatusBar(false)
	listModel.SetShowTitle(true)
	listModel.Styles.Title = tui.TitleStyle
	listModel.Styles.TitleBar = tui.TitleStyle
	listModel.Styles.PaginationStyle = tui.PaginationStyle
	listModel.Styles.HelpStyle = tui.HelpStyle
	listModel.KeyMap = tui.TranslatedListKeyMap()
	listModel.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{toggleBinding(), tui.Accept(), tui.QuitWithEsc()}
	}
	listModel.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{toggleBinding(), tui.Accept(), tui.QuitWithEsc()}
	}

	return ReleaseTypesModel{
		list:     listModel,
		selected: selected,
	}
}

func (m ReleaseTypesModel) Init() tea.Cmd {
	return nil
}

func (m ReleaseTypesModel) Update(msg tea.Msg) (ReleaseTypesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case " ":
			m.toggleSelected()
		case "enter":
			values := m.values()
			if len(values) == 0 {
				m.error = errors.New("release types cannot be empty")
				return m, nil
			}
			m.Value = values
			return m, m.releaseTypesSelected()
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ReleaseTypesModel) View() string {
	if m.error != nil {
		return m.list.View() + "\n" + tui.ErrorStyle.Render(m.error.Error())
	}

	return m.list.View()
}

func (m ReleaseTypesModel) releaseTypesSelected() tea.Cmd {
	return func() tea.Msg {
		return ReleaseTypesSelectedMessage{ReleaseTypes: m.Value}
	}
}

func (m ReleaseTypesModel) toggleSelected() {
	item, ok := m.list.SelectedItem().(releaseTypeItem)
	if !ok {
		return
	}

	current := m.selected[item.value]
	m.selected[item.value] = !current
}

func (m ReleaseTypesModel) values() []models.ReleaseType {
	values := make([]models.ReleaseType, 0)
	for _, releaseType := range models.AllReleaseTypes() {
		if m.selected[releaseType] {
			values = append(values, releaseType)
		}
	}
	return values
}

func formatReleaseTypes(values []models.ReleaseType) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, string(value))
	}
	return strings.Join(parts, ", ")
}

type releaseTypeDelegate struct{}

func (d releaseTypeDelegate) Height() int                             { return 1 }
func (d releaseTypeDelegate) Spacing() int                            { return 0 }
func (d releaseTypeDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d releaseTypeDelegate) Render(w io.Writer, m list.Model, itemIndex int, listItem list.Item) {
	item, ok := listItem.(releaseTypeItem)
	if !ok {
		return
	}

	icon := " "
	if item.selected[item.value] {
		icon = "✓"
	}

	itemLine := fmt.Sprintf("%s %s", icon, item.value)

	if itemIndex == m.Index() {
		if _, err := fmt.Fprint(w, tui.SelectedItemStyle.Render("❯ "+itemLine)); err != nil {
			return
		}
		return
	}

	if _, err := fmt.Fprint(w, tui.ItemStyle.Render(itemLine)); err != nil {
		return
	}
}

type releaseTypeItem struct {
	value    models.ReleaseType
	selected map[models.ReleaseType]bool
}

func (i releaseTypeItem) FilterValue() string { return string(i.value) }

func toggleBinding() key.Binding {
	return key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle"),
	)
}
