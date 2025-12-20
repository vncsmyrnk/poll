package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewHandler(pollHandler *PollHandler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Route("/api", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("welcome"))
		})

		r.Route("/polls", func(r chi.Router) {
			r.Post("/", pollHandler.CreatePoll)
			r.Get("/{id}", pollHandler.GetPoll)
		})

		r.Route("/votes", func(r chi.Router) {
		})
	})

	return r
}
