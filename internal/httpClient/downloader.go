package httpClient

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/fileutils"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/spf13/afero"
	"io"
	"net/http"
)

type progressWriter struct {
	total      int
	downloaded int
	file       afero.File
	reader     io.Reader
	onProgress func(float64)
}

type progressMsg float64

type progressErrMsg struct{ err error }

func (pw *progressWriter) Write(p []byte) (int, error) {
	pw.downloaded += len(p)
	if pw.total > 0 && pw.onProgress != nil {
		pw.onProgress(float64(pw.downloaded) / float64(pw.total))
	}
	return len(p), nil
}

type Sender interface {
	Send(msg tea.Msg)
}

func DownloadFile(url string, filepath string, client Doer, program Sender, filesystem ...afero.Fs) error {
	region := perf.StartRegionWithDetails("download-file", &perf.PerformanceDetails{
		"file": url,
	})
	defer region.End()

	fs := fileutils.InitFilesystem(filesystem...)
	request, _ := http.NewRequest("GET", url, nil)
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	defer response.Body.Close()

	file, err := fs.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	pw := &progressWriter{
		total:  int(response.ContentLength),
		file:   file,
		reader: response.Body,
		onProgress: func(ratio float64) {
			program.Send(progressMsg(ratio))
		},
	}

	_, err = io.Copy(pw.file, io.TeeReader(pw.reader, pw))
	if err != nil {
		err2 := fmt.Errorf("failed to write file: %w", err)
		program.Send(progressErrMsg{err2})
		return err2
	}

	return nil
}
