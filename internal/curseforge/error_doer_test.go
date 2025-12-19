package curseforge

import "net/http"

type errorDoer struct {
	err error
}

func (d errorDoer) Do(_ *http.Request) (*http.Response, error) {
	return nil, d.err
}
