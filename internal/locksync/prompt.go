package locksync

import (
	"errors"
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type lockSyncPromptInput struct {
	listView string
	runTea   func(tea.Model, ...tea.ProgramOption) (tea.Model, error)
	in       io.Reader
	out      io.Writer
}

type lockSyncPromptOutcome struct {
	policy   Policy
	canceled bool
}

func runLockSyncPrompt(input lockSyncPromptInput) (lockSyncPromptOutcome, error) {
	if input.runTea == nil {
		return lockSyncPromptOutcome{}, errors.New("missing lock sync prompt runner")
	}
	model := newLockSyncPromptModel(input.listView)
	options := view.ProgramOptions(input.in, input.out)
	result, err := input.runTea(model, options...)
	if err != nil {
		return lockSyncPromptOutcome{}, err
	}
	return lockSyncPromptResult(result)
}

func lockSyncPromptResult(result tea.Model) (lockSyncPromptOutcome, error) {
	switch typed := result.(type) {
	case lockSyncPromptModel:
		return lockSyncPromptOutcome{
			policy:   typed.policy,
			canceled: typed.canceled,
		}, nil
	case *lockSyncPromptModel:
		return lockSyncPromptOutcome{
			policy:   typed.policy,
			canceled: typed.canceled,
		}, nil
	default:
		return lockSyncPromptOutcome{}, errors.New("unexpected lock sync prompt model")
	}
}

type lockSyncPromptModel struct {
	listView string
	list     list.Model
	canceled bool
	policy   Policy
	answered bool
}

func newLockSyncPromptModel(listView string) lockSyncPromptModel {
	items := []list.Item{
		lockSyncPolicyItem{
			label:  i18n.T("cmd.lock_sync.option.add", nil),
			policy: PolicyAdd,
		},
		lockSyncPolicyItem{
			label:  i18n.T("cmd.lock_sync.option.delete", nil),
			policy: PolicyDelete,
		},
		lockSyncPolicyItem{
			label:  i18n.T("cmd.lock_sync.option.ignore", nil),
			policy: PolicyIgnore,
		},
		lockSyncPolicyItem{
			label:  i18n.T("cmd.lock_sync.answer.skip", nil),
			policy: PolicySkip,
		},
	}

	listModel := list.New(items, lockSyncPolicyDelegate{}, 80, len(items)+5)
	listModel.Title = lockSyncQuestionLine()
	listModel.SetFilteringEnabled(false)
	listModel.SetShowStatusBar(false)
	listModel.SetShowPagination(false)
	listModel.SetShowHelp(true)
	listModel.SetShowTitle(true)
	listModel.Styles.Title = view.TitleStyle
	listModel.Styles.TitleBar = view.TitleStyle
	listModel.Styles.PaginationStyle = view.PaginationStyle
	listModel.Styles.HelpStyle = view.HelpStyle
	listModel.KeyMap = view.TranslatedListKeyMap()
	listModel.Select(0)

	return lockSyncPromptModel{
		listView: listView,
		list:     listModel,
	}
}

func (model lockSyncPromptModel) Init() tea.Cmd {
	return nil
}

func (model lockSyncPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		model.list.SetWidth(typed.Width)
		return model, nil
	case tea.KeyMsg:
		if typed.String() == "ctrl+c" || typed.String() == "esc" || typed.String() == "q" {
			model.canceled = true
			return model, tea.Quit
		}
		if typed.String() == "enter" {
			item, ok := model.list.SelectedItem().(lockSyncPolicyItem)
			if ok {
				model.policy = item.policy
				model.answered = true
				return model, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	model.list, cmd = model.list.Update(msg)
	return model, cmd
}

func (model lockSyncPromptModel) View() string {
	if model.answered {
		return joinPromptLines(model.listView, lockSyncAnswerLine(model.policy))
	}
	return joinPromptLines(model.listView, model.list.View())
}

func joinPromptLines(listView string, promptView string) string {
	if listView == "" {
		return promptView
	}
	if promptView == "" {
		return listView
	}
	return listView + "\n\n" + promptView
}

type lockSyncPolicyItem struct {
	label  string
	policy Policy
}

func (item lockSyncPolicyItem) FilterValue() string { return "" }

type lockSyncPolicyDelegate struct{}

func (delegate lockSyncPolicyDelegate) Height() int                             { return 1 }
func (delegate lockSyncPolicyDelegate) Spacing() int                            { return 0 }
func (delegate lockSyncPolicyDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (delegate lockSyncPolicyDelegate) Render(writer io.Writer, listModel list.Model, itemIndex int, listItem list.Item) {
	item, ok := listItem.(lockSyncPolicyItem)
	if !ok {
		return
	}

	line := item.label
	if itemIndex == listModel.Index() {
		if _, err := fmt.Fprint(writer, view.SelectedItemStyle.Render(lockSyncPointer()+line)); err != nil {
			return
		}
		return
	}

	if _, err := fmt.Fprint(writer, view.ItemStyle.Render(line)); err != nil {
		return
	}
}

func lockSyncPointer() string {
	if view.SupportsUnicode() {
		return "\u276F "
	}
	return "> "
}

func lockSyncQuestionLine() string {
	return view.QuestionStyle.Render("? ") + view.TitleStyle.Render(i18n.T("cmd.lock_sync.prompt", nil))
}

func lockSyncAnswerLine(policy Policy) string {
	label, ok := lockSyncAnswerLabel(policy)
	question := lockSyncQuestionLine()
	if !ok {
		return question
	}
	return question + " " + view.SelectedItemStyle.Render(label)
}

func lockSyncAnswerLabel(policy Policy) (string, bool) {
	switch policy {
	case PolicyAdd:
		return i18n.T("cmd.lock_sync.answer.add", nil), true
	case PolicyDelete:
		return i18n.T("cmd.lock_sync.answer.delete", nil), true
	case PolicyIgnore:
		return i18n.T("cmd.lock_sync.answer.ignore", nil), true
	case PolicySkip:
		return i18n.T("cmd.lock_sync.answer.skip", nil), true
	default:
		return "", false
	}
}
