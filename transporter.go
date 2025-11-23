package twitchgo

import (
	"bytes"
	"io"
	"net/http"

	"github.com/nicklaw5/helix/v2"
)

type HelixRefreshTransport struct {
	Base   http.RoundTripper
	Client *helix.Client
	Event  EventEngine
}

func (t *HelixRefreshTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	rt := t.Base
	if rt == nil {
		rt = http.DefaultTransport
	}

	resp, err := rt.RoundTrip(r)
	if err != nil {
		return resp, err
	}

	if resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	}

	refresh := t.Client.GetRefreshToken()
	if refresh == "" {
		return resp, err
	}

	newTokens, refreshErr := t.Client.RefreshUserAccessToken(refresh)
	if refreshErr != nil {
		return resp, refreshErr
	}

	t.Client.SetUserAccessToken(newTokens.Data.AccessToken)
	t.Client.SetRefreshToken(newTokens.Data.RefreshToken)
	t.Event.OnClientRefresh(r.Context(), t.Client)

	retryReq := cloneRequest(r)
	return rt.RoundTrip(retryReq)
}

func cloneRequest(r *http.Request) *http.Request {
	c := r.Clone(r.Context())

	if r.Body != nil {
		data, _ := io.ReadAll(r.Body)
		r.Body.Close()
		r.Body = io.NopCloser(bytes.NewBuffer(data))
		c.Body = io.NopCloser(bytes.NewBuffer(data))
	}
	return c
}
