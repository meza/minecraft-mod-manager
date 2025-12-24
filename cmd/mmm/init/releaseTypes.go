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

func (model ReleaseTypesModel) Init() tea.Cmd {
	return nil
}

func (model ReleaseTypesModel) Update(msg tea.Msg) (ReleaseTypesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		model.list.SetWidth(msg.Width)
		return model, nil
	case tea.KeyMsg:
		switch msg.String() {
		case " ":
			model.toggleSelected()
		case "enter":
			values := model.values()
			if len(values) == 0 {
				model.error = errors.New("release types cannot be empty")
				return model, nil
			}
			model.Value = values
			return model, model.releaseTypesSelected()
		}
	}

	var cmd tea.Cmd
	model.list, cmd = model.list.Update(msg)
	return model, cmd
}

func (model ReleaseTypesModel) View() string {
	if model.error != nil {
		return model.list.View() + "\n" + tui.ErrorStyle.Render(model.error.Error())
	}

	return model.list.View()
}

func (model ReleaseTypesModel) releaseTypesSelected() tea.Cmd {
	return func() tea.Msg {
		return ReleaseTypesSelectedMessage{ReleaseTypes: model.Value}
	}
}

func (model ReleaseTypesModel) toggleSelected() {
	item, ok := model.list.SelectedItem().(releaseTypeItem)
	if !ok {
		return
	}

	current := model.selected[item.value]
	model.selected[item.value] = !current
}

func (model ReleaseTypesModel) values() []models.ReleaseType {
	values := make([]models.ReleaseType, 0)
	for _, releaseType := range models.AllReleaseTypes() {
		if model.selected[releaseType] {
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

func (delegate releaseTypeDelegate) Height() int                             { return 1 }
func (delegate releaseTypeDelegate) Spacing() int                            { return 0 }
func (delegate releaseTypeDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (delegate releaseTypeDelegate) Render(w io.Writer, listModel list.Model, itemIndex int, listItem list.Item) {
	item, ok := listItem.(releaseTypeItem)
	if !ok {
		return
	}

	icon := " "
	if item.selected[item.value] {
		icon = "✓"
	}

	itemLine := fmt.Sprintf("%s %s", icon, item.value)

	if itemIndex == listModel.Index() {
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

func (item releaseTypeItem) FilterValue() string { return string(item.value) }

func toggleBinding() key.Binding {
	return key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle"),
	)
}
