package twitchgo

import (
	"os"

	"github.com/Etwodev/twitchgo/pkg/helpers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (b *Bot) Routes(m *chi.Mux) {
	m.Use(middleware.RequestID)
	m.Use(middleware.RealIP)
	m.Use(middleware.Recoverer)

	m.Post("/webhook/callback", b.Handle)
	m.Route("/auth", func(r chi.Router) {
		r.Get("/login", HandleLogin)
		r.Get("/callback", helpers.SimpleBasicAuth(
			os.Getenv("CALLBACK_USER"),
			os.Getenv("CALLBACK_PASS"),
			b.HandleCallback,
		))
	})

	m.Get("/healthcheck", HandleHealthCheck)
}
