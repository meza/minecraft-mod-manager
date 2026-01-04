package init

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type LoaderSelectedMessage struct {
	Loader models.Loader
}

type LoaderModel struct {
	tea.Model
	list  list.Model
	Value models.Loader
}

func (model LoaderModel) Init() tea.Cmd {
	return nil
}

func (model LoaderModel) Update(msg tea.Msg) (LoaderModel, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		model.list.SetWidth(msg.Width)
		return model, nil
	case tea.KeyMsg:
		if keypress := msg.String(); keypress == "enter" {
			item, ok := model.list.SelectedItem().(loaderType)
			if ok {
				model.Value = models.Loader(item)
				cmds = append(cmds, model.loaderSelected())
			}
		}
	}

	var cmd tea.Cmd
	model.list, cmd = model.list.Update(msg)
	cmds = append(cmds, cmd)
	return model, tea.Batch(cmds...)
}

func (model LoaderModel) View() string {
	if model.Value != "" {
		return fmt.Sprintf("%s %s", model.Title(), view.SelectedItemStyle.Render(string(model.Value)))
	}
	return model.list.View()
}

func (model LoaderModel) Title() string {
	return model.list.Title
}

func (model LoaderModel) loaderSelected() tea.Cmd {
	return func() tea.Msg {
		return LoaderSelectedMessage{Loader: model.Value}
	}
}

type itemDelegate struct{}

func (delegate itemDelegate) Height() int                             { return 1 }
func (delegate itemDelegate) Spacing() int                            { return 0 }
func (delegate itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (delegate itemDelegate) Render(w io.Writer, listModel list.Model, itemIndex int, listItem list.Item) {
	item, ok := listItem.(loaderType)
	if !ok {
		return
	}

	itemLine := string(item)

	if itemIndex == listModel.Index() {
		if _, err := fmt.Fprint(w, view.SelectedItemStyle.Render(focusedLoaderPrefix()+itemLine)); err != nil {
			return
		}
		return
	}

	if _, err := fmt.Fprint(w, view.ItemStyle.Render(itemLine)); err != nil {
		return
	}
}

type loaderType string

func (item loaderType) FilterValue() string { return "" }

func NewLoaderModel(loader string) LoaderModel {
	loaderOptions := models.AllLoaders()
	items := make([]list.Item, 0)

	for _, loader := range loaderOptions {
		items = append(items, loaderType(loader))
	}

	listModel := list.New(items, itemDelegate{}, 0, 14)
	listModel.Title = view.QuestionStyle.Render("? ") + view.TitleStyle.Render(i18n.T("cmd.init.prompt.loader.question", nil))
	listModel.SetShowStatusBar(false)
	listModel.SetShowTitle(true)
	listModel.Styles.Title = view.TitleStyle
	listModel.Styles.TitleBar = view.TitleStyle
	listModel.Styles.PaginationStyle = view.PaginationStyle
	listModel.Styles.HelpStyle = view.HelpStyle
	listModel.KeyMap = view.TranslatedListKeyMap()

	model := LoaderModel{
		list: listModel,
	}

	loaderVal := models.Loader(loader)
	if isValidLoader(loaderVal) {
		model.Value = loaderVal
		for idx, item := range items {
			typedItem, ok := item.(loaderType)
			if ok && typedItem == loaderType(loaderVal) {
				model.list.Select(idx)
				break
			}
		}
	}

	return model
}

func focusedLoaderPrefix() string {
	prefix := "> "
	if view.SupportsUnicode() {
		prefix = "\u276F "
	}
	return prefix
}

func isValidLoader(loader models.Loader) bool {
	for _, l := range models.AllLoaders() {
		if l == loader {
			return true
		}
	}
	return false
}
