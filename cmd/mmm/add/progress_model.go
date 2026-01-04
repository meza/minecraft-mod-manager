package add

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type downloadProgressStatus int

const (
	progressStatusStarted downloadProgressStatus = iota
	progressStatusInProgress
	progressStatusCompleted
	progressStatusFailed
)

type downloadProgressModel struct {
	modName    string
	modID      string
	platform   models.Platform
	colorMode  view.ColorMode
	download   func(httpclient.Sender) (modinstall.EnsureResult, error)
	progress   progress.Model
	status     downloadProgressStatus
	ratio      float64
	downloaded int64
	total      int64
	err        error
	sender     httpclient.Sender
	result     modinstall.EnsureResult
}

type downloadFinishedMsg struct {
	result modinstall.EnsureResult
}

type downloadFailedMsg struct {
	err error
}

type progressRunner func(model *downloadProgressModel, options ...tea.ProgramOption) (tea.Model, error)

var runProgressProgram progressRunner = defaultRunProgressProgram

func defaultRunProgressProgram(model *downloadProgressModel, options ...tea.ProgramOption) (tea.Model, error) {
	program := tea.NewProgram(model, options...)
	model.sender = program
	return program.Run()
}

func newDownloadProgressModel(modName string, modID string, platformValue models.Platform, colorMode view.ColorMode, download func(httpclient.Sender) (modinstall.EnsureResult, error)) *downloadProgressModel {
	bar := progress.New(
		progress.WithWidth(10),
		progress.WithoutPercentage(),
	)
	if !view.SupportsUnicode() {
		bar = progress.New(
			progress.WithWidth(10),
			progress.WithoutPercentage(),
			progress.WithFillCharacters('#', '-'),
		)
	}

	return &downloadProgressModel{
		modName:   modName,
		modID:     modID,
		platform:  platformValue,
		colorMode: colorMode,
		download:  download,
		progress:  bar,
		status:    progressStatusStarted,
	}
}

func (model *downloadProgressModel) Init() tea.Cmd {
	return model.startDownloadCmd()
}

func (model *downloadProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case httpclient.DownloadProgressMsg:
		model.status = progressStatusInProgress
		model.ratio = typed.Ratio
		model.downloaded = typed.Downloaded
		model.total = typed.Total
		return model, nil
	case httpclient.DownloadProgressErrMsg:
		if model.err == nil {
			model.err = typed.Err
		}
		return model, nil
	case downloadFinishedMsg:
		model.status = progressStatusCompleted
		model.result = typed.result
		return model, tea.Quit
	case downloadFailedMsg:
		model.status = progressStatusFailed
		model.err = typed.err
		return model, tea.Quit
	default:
		return model, nil
	}
}

func (model *downloadProgressModel) View() string {
	switch model.status {
	case progressStatusStarted:
		return model.startedLine()
	case progressStatusInProgress:
		return model.progressLine()
	default:
		return ""
	}
}

func (model *downloadProgressModel) startDownloadCmd() tea.Cmd {
	return func() tea.Msg {
		if model.download == nil {
			return downloadFailedMsg{err: errors.New("missing download handler")}
		}
		if model.sender == nil {
			return downloadFailedMsg{err: errors.New("missing bubble tea sender")}
		}
		result, err := model.download(model.sender)
		if err != nil {
			return downloadFailedMsg{err: err}
		}
		return downloadFinishedMsg{result: result}
	}
}

func (model *downloadProgressModel) startedLine() string {
	icon := view.RenderIfColorEnabled(model.colorMode, view.QuestionStyle, preparingIcon())
	message := i18nLine("cmd.add.progress.start", model.modName, model.modID, model.platform, model.colorMode)
	return fmt.Sprintf("%s %s", icon, message)
}

func (model *downloadProgressModel) progressLine() string {
	icon := view.RenderIfColorEnabled(model.colorMode, view.QuestionStyle, downloadingIcon())
	message := i18nLine("cmd.add.progress.downloading", model.modName, model.modID, model.platform, model.colorMode)
	barLine := model.progress.ViewAs(model.ratio)
	if !view.SupportsUnicode() {
		barLine = "[" + barLine + "]"
	}
	percentLine := progressPercentLine(model.ratio, model.downloaded, model.total)
	return strings.Join([]string{fmt.Sprintf("%s %s", icon, message), barLine, percentLine}, "\n")
}

func i18nLine(key string, name string, id string, platformValue models.Platform, colorMode view.ColorMode) string {
	idValue := view.RenderIfColorEnabled(colorMode, view.ParenStyle, id)
	platformText := view.RenderIfColorEnabled(colorMode, view.ParenStyle, string(platformValue))
	return i18n.T(key, &i18n.Tvars{
		Data: &i18n.TData{
			"name":     name,
			"id":       idValue,
			"platform": platformText,
		},
	})
}

func progressPercentLine(ratio float64, downloaded int64, total int64) string {
	if total <= 0 {
		return fmt.Sprintf("%d%%", percentFromRatio(ratio))
	}
	return fmt.Sprintf("%d%% (%s / %s)", percentFromRatio(ratio), formatBytes(downloaded), formatBytes(total))
}

func percentFromRatio(ratio float64) int {
	if ratio <= 0 {
		return 0
	}
	if ratio >= 1 {
		return 100
	}
	return int(math.Round(ratio * 100))
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	value := float64(bytes)
	suffixes := []string{"KB", "MB", "GB", "TB"}
	for _, suffix := range suffixes {
		value = value / unit
		if value < unit {
			return fmt.Sprintf("%d %s", int64(math.Round(value)), suffix)
		}
	}
	return fmt.Sprintf("%d TB", int64(math.Round(value)))
}

func preparingIcon() string {
	if view.SupportsUnicode() {
		return "\u23F3"
	}
	return "[~]"
}

func downloadingIcon() string {
	if view.SupportsUnicode() {
		return "\u2B07\uFE0F"
	}
	return "->"
}
