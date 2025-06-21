package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"io"
	"os"
)

type LoaderSelectedMessage struct {
	Loader models.Loader
}

type LoaderModel struct {
	tea.Model
	list  list.Model
	Value models.Loader
}

func (m LoaderModel) Init() tea.Cmd {
	return nil
}

func (m LoaderModel) Update(msg tea.Msg) (LoaderModel, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "enter":
			i, ok := m.list.SelectedItem().(loaderType)
			if ok {
				m.Value = models.Loader(i)
				cmds = append(cmds, m.loaderSelected())
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m LoaderModel) View() string {

	if m.Value != "" {
		return fmt.Sprintf("%s %s", m.Title(), SelectedItemStyle.Render(string(m.Value)))
	}
	return m.list.View()
}

func (m LoaderModel) Title() string {
	return m.list.Title
}

func (m LoaderModel) loaderSelected() tea.Cmd {
	return func() tea.Msg {
		return LoaderSelectedMessage{Loader: m.Value}
	}
}

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, itemIndex int, listItem list.Item) {
	item, ok := listItem.(loaderType)
	if !ok {
		return
	}

	itemLine := fmt.Sprintf("%s", item)

	if itemIndex == m.Index() {
		fmt.Fprint(w, SelectedItemStyle.Render("❯ "+itemLine))
		return
	}

	fmt.Fprint(w, ItemStyle.Render(itemLine))
}

type loaderType string

func (i loaderType) FilterValue() string { return "" }

func NewLoaderModel(loader string) LoaderModel {
	width, _, _ := term.GetSize(os.Stdout.Fd())

	loaderOptions := models.AllLoaders()
	items := make([]list.Item, 0)

	for _, loader := range loaderOptions {
		items = append(items, loaderType(loader))
	}

	listModel := list.New(items, itemDelegate{}, width, 14)
	listModel.Title = QuestionStyle.Render("? ") + TitleStyle.Render(i18n.T("cmd.init.tui.loader.question"))
	listModel.SetShowStatusBar(false)
	listModel.SetShowTitle(true)
	listModel.Styles.Title = TitleStyle
	listModel.Styles.TitleBar = TitleStyle
	listModel.Styles.PaginationStyle = PaginationStyle
	listModel.Styles.HelpStyle = HelpStyle
	listModel.KeyMap = TranslatedListKeyMap()

	model := LoaderModel{
		list: listModel,
	}

	loaderVal := models.Loader(loader)
	if isValidLoader(loaderVal) {
		model.Value = loaderVal
	}

	return model
}

func isValidLoader(loader models.Loader) bool {
	for _, l := range models.AllLoaders() {
		if l == loader {
			return true
		}
	}
	return false
}
