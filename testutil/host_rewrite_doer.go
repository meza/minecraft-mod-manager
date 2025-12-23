// Package testutil holds shared test helpers.
package testutil

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
)

type HostRewriteDoer struct {
	base *url.URL
	next httpclient.Doer
}

func NewHostRewriteDoer(serverURL string, next httpclient.Doer) (*HostRewriteDoer, error) {
	if next == nil {
		return nil, fmt.Errorf("next doer is nil")
	}

	base, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("parse server url: %w", err)
	}
	if base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("server url must include scheme and host")
	}

	return &HostRewriteDoer{
		base: base,
		next: next,
	}, nil
}

func MustNewHostRewriteDoer(serverURL string, next httpclient.Doer) *HostRewriteDoer {
	doer, err := NewHostRewriteDoer(serverURL, next)
	if err != nil {
		panic(err)
	}
	return doer
}

func (d *HostRewriteDoer) Do(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = d.base.Scheme
	cloned.URL.Host = d.base.Host
	cloned.Host = d.base.Host
	return d.next.Do(cloned)
}
