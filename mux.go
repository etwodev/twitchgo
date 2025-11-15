package twitchgo

import "github.com/go-chi/chi/v5"

// handler creates and returns the root chi.Mux router for the server.
//
// It initializes the mux with middleware and routers previously loaded.
//
// Example:
//
//	mux := srv.handler()
func (b *Bot) handler() *chi.Mux {
	m := chi.NewMux()
	b.Routes(m)
	return m
}
