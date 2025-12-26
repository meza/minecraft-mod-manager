package httpclient

import (
	"net/url"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
)

type InvalidDownloadURLError struct {
	URL string
}

func (err InvalidDownloadURLError) Error() string {
	return i18n.T("error.download_url_invalid", i18n.Tvars{
		Data: &i18n.TData{
			"url": err.URL,
		},
	})
}

type InsecureDownloadURLError struct {
	URL string
}

func (err InsecureDownloadURLError) Error() string {
	return i18n.T("error.download_url_insecure", i18n.Tvars{
		Data: &i18n.TData{
			"url": err.URL,
		},
	})
}

type UntrustedDownloadHostError struct {
	Host string
	URL  string
}

func (err UntrustedDownloadHostError) Error() string {
	return i18n.T("error.download_url_untrusted_host", i18n.Tvars{
		Data: &i18n.TData{
			"host": err.Host,
			"url":  err.URL,
		},
	})
}

var allowedDownloadHosts = map[string]struct{}{
	"cdn.modrinth.com":   {},
	"edge.forgecdn.net":  {},
	"media.forgecdn.net": {},
}

func validateDownloadURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed == nil {
		return nil, InvalidDownloadURLError{URL: rawURL}
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, InvalidDownloadURLError{URL: rawURL}
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return nil, InsecureDownloadURLError{URL: rawURL}
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return nil, InvalidDownloadURLError{URL: rawURL}
	}
	if _, ok := allowedDownloadHosts[host]; !ok {
		return nil, UntrustedDownloadHostError{Host: host, URL: rawURL}
	}
	return parsed, nil
}
