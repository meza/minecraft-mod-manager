// Package httpclient provides shared HTTP primitives.
package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/fileutils"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/spf13/afero"
	"go.opentelemetry.io/otel/attribute"
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

func DownloadFile(ctx context.Context, url string, filepath string, client Doer, program Sender, filesystemOverrides ...afero.Fs) (returnErr error) {
	_, span := perf.StartSpan(ctx, "io.download.file",
		perf.WithAttributes(
			attribute.String("url", url),
			attribute.String("path", filepath),
		),
	)
	defer span.End()

	filesystem := fileutils.InitFilesystem(filesystemOverrides...)
	request, cancel, err := buildDownloadRequest(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to build download request: %w", err)
	}
	defer cancel()
	response, err := client.Do(request)
	if err != nil {
		return WrapTimeoutError(fmt.Errorf("failed to download file: %w", err))
	}

	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && returnErr == nil {
			returnErr = closeErr
		}
	}()

	if err := validateDownloadResponse(response); err != nil {
		return err
	}

	file, err := createDownloadFile(filesystem, filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && returnErr == nil {
			returnErr = closeErr
		}
	}()

	progressWriter := buildProgressWriter(response, file, program, span)
	if err := copyDownload(progressWriter); err != nil {
		return handleDownloadWriteError(err, program, filesystem, filepath)
	}

	return nil
}

func buildDownloadRequest(ctx context.Context, url string) (*http.Request, func(), error) {
	downloadCtx, cancel := WithDownloadTimeout(ctx)
	request, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, url, http.NoBody)
	if err != nil {
		cancel()
		return nil, func() {}, err
	}
	return request, cancel, nil
}

func validateDownloadResponse(response *http.Response) error {
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("download request failed with status %d", response.StatusCode)
	}
	return nil
}

func createDownloadFile(filesystem afero.Fs, path string) (afero.File, error) {
	return filesystem.Create(path)
}

func buildProgressWriter(response *http.Response, file afero.File, program Sender, span *perf.Span) *progressWriter {
	progressWriter := &progressWriter{
		total:  int(response.ContentLength),
		file:   file,
		reader: response.Body,
		onProgress: func(ratio float64) {
			program.Send(progressMsg(ratio))
		},
	}
	if progressWriter.total > 0 {
		span.SetAttributes(attribute.Int64("bytes", int64(progressWriter.total)))
	}
	return progressWriter
}

func copyDownload(progressWriter *progressWriter) error {
	_, err := io.Copy(progressWriter.file, io.TeeReader(progressWriter.reader, progressWriter))
	if err != nil {
		return WrapTimeoutError(fmt.Errorf("failed to write file: %w", err))
	}
	return nil
}

func handleDownloadWriteError(err error, program Sender, filesystem afero.Fs, path string) error {
	program.Send(progressErrMsg{err})
	if removeErr := filesystem.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return errors.Join(err, fmt.Errorf("failed to remove partial file: %w", removeErr))
	}
	return err
}
