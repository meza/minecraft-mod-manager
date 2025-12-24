package curseforge

import "net/http"

type errorDoer struct {
	err error
}

func (doer errorDoer) Do(_ *http.Request) (*http.Response, error) {
	return nil, doer.err
}
