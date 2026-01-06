package change

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type changePolicyPromptModel struct {
	list list.Model
}

func newChangePolicyPromptModel() *changePolicyPromptModel {
	items := []list.Item{
		changeForcePolicyItem{
			label:  i18n.T("cmd.change.force_policy.option.keep", nil),
			policy: changeForcePolicyKeepConfig,
		},
		changeForcePolicyItem{
			label:  i18n.T("cmd.change.force_policy.option.prune", nil),
			policy: changeForcePolicyPruneConfig,
		},
		changeForcePolicyItem{
			label:  i18n.T("cmd.change.force_policy.option.disable", nil),
			policy: changeForcePolicyDisableSkipped,
		},
	}

	listModel := list.New(items, changeForcePolicyDelegate{}, 80, len(items)+5)
	listModel.Title = forcePolicyQuestionLine()
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

	return &changePolicyPromptModel{list: listModel}
}

func (model changePolicyPromptModel) Update(msg tea.Msg) (changePolicyPromptModel, tea.Cmd) {
	var cmds []tea.Cmd
	keyMsg, ok := msg.(tea.KeyMsg)
	if ok && keyMsg.String() == "enter" {
		item, ok := model.list.SelectedItem().(changeForcePolicyItem)
		if ok {
			cmds = append(cmds, policySelected(item.policy))
		}
	}

	var cmd tea.Cmd
	model.list, cmd = model.list.Update(msg)
	cmds = append(cmds, cmd)
	return model, tea.Batch(cmds...)
}

func (model changePolicyPromptModel) View() string {
	return model.list.View()
}

func (model *changePolicyPromptModel) SetWidth(width int) {
	if model == nil {
		return
	}
	model.list.SetWidth(width)
}

type changeForcePolicyItem struct {
	label  string
	policy changeForcePolicy
}

func (item changeForcePolicyItem) FilterValue() string { return "" }

type changeForcePolicyDelegate struct{}

func (delegate changeForcePolicyDelegate) Height() int                             { return 1 }
func (delegate changeForcePolicyDelegate) Spacing() int                            { return 0 }
func (delegate changeForcePolicyDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (delegate changeForcePolicyDelegate) Render(writer io.Writer, listModel list.Model, itemIndex int, listItem list.Item) {
	item, ok := listItem.(changeForcePolicyItem)
	if !ok {
		return
	}

	line := item.label
	if itemIndex == listModel.Index() {
		if _, err := fmt.Fprint(writer, view.SelectedItemStyle.Render(forcePolicyPointer()+line)); err != nil {
			return
		}
		return
	}

	if _, err := fmt.Fprint(writer, view.ItemStyle.Render(line)); err != nil {
		return
	}
}

func forcePolicyPointer() string {
	if view.SupportsUnicode() {
		return "\u276F "
	}
	return "> "
}

func policySelected(policy changeForcePolicy) tea.Cmd {
	return func() tea.Msg {
		return changePolicySelectedMsg{policy: policy}
	}
}

func forcePolicyAnswerLine(policy changeForcePolicy) string {
	label, ok := forcePolicyAnswerLabel(policy)
	question := forcePolicyQuestionLine()
	if !ok {
		return question
	}
	return question + " " + view.SelectedItemStyle.Render(label)
}

func forcePolicyQuestionLine() string {
	return view.QuestionStyle.Render("? ") + view.TitleStyle.Render(i18n.T("cmd.change.force_policy.prompt", nil))
}

func forcePolicyLabel(policy changeForcePolicy) (string, bool) {
	switch policy {
	case changeForcePolicyKeepConfig:
		return i18n.T("cmd.change.force_policy.option.keep", nil), true
	case changeForcePolicyPruneConfig:
		return i18n.T("cmd.change.force_policy.option.prune", nil), true
	case changeForcePolicyDisableSkipped:
		return i18n.T("cmd.change.force_policy.option.disable", nil), true
	default:
		return "", false
	}
}

func forcePolicyAnswerLabel(policy changeForcePolicy) (string, bool) {
	switch policy {
	case changeForcePolicyKeepConfig:
		return i18n.T("cmd.change.force_policy.answer.keep", nil), true
	case changeForcePolicyPruneConfig:
		return i18n.T("cmd.change.force_policy.answer.prune", nil), true
	case changeForcePolicyDisableSkipped:
		return i18n.T("cmd.change.force_policy.answer.disable", nil), true
	default:
		return "", false
	}
}
