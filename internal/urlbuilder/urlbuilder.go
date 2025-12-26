package urlbuilder

import (
	"net/url"
	"path"
	"strings"
)

func JoinEscapedPath(base *url.URL, segments ...string) *url.URL {
	joined := *base

	basePath := strings.TrimSuffix(base.Path, "/")
	if basePath == "" {
		basePath = "/"
	}
	baseRawPath := strings.TrimSuffix(base.RawPath, "/")
	if baseRawPath == "" {
		baseRawPath = basePath
	}

	rawSegments := make([]string, 0, len(segments)+1)
	escapedSegments := make([]string, 0, len(segments)+1)
	rawSegments = append(rawSegments, basePath)
	escapedSegments = append(escapedSegments, baseRawPath)

	for _, segment := range segments {
		rawSegments = append(rawSegments, segment)
		escapedSegments = append(escapedSegments, url.PathEscape(segment))
	}

	joined.Path = path.Join(rawSegments...)
	escapedPath := path.Join(escapedSegments...)
	if joined.Path != escapedPath {
		joined.RawPath = escapedPath
	} else {
		joined.RawPath = ""
	}

	return &joined
}
