package add

import (
	"errors"

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
	bar := view.NewProgressBar()

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
	message := i18nLine("cmd.add.progress.start", model.modName, model.modID, model.platform, model.colorMode)
	return view.RenderModItemLine(view.ModItemLine{
		Label:  message,
		Status: view.ModItemStatusPending,
	}, model.colorMode)
}

func (model *downloadProgressModel) progressLine() string {
	message := view.RenderModLabel(model.colorMode, model.modName, model.modID, string(model.platform))
	return view.RenderModItemLine(view.ModItemLine{
		Label:  message,
		Status: view.ModItemStatusDownloading,
		Progress: &view.ProgressDetails{
			Bar:        model.progress,
			Ratio:      model.ratio,
			Downloaded: model.downloaded,
			Total:      model.total,
		},
	}, model.colorMode)
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
