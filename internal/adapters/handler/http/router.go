package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewHandler(pollHandler *PollHandler, voteHandler *VoteHandler, authHandler *AuthHandler, userHandler *UserHandler, allowedOrigins []string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(NewCorsMiddleware(allowedOrigins))

	r.Route("/api", func(r chi.Router) {
		r.With(AuthMiddleware).Get("/me", userHandler.GetMe)

		r.Route("/polls", func(r chi.Router) {
			r.Get("/", pollHandler.ListPolls)
			r.Post("/", pollHandler.CreatePoll)
			r.Get("/{id}", pollHandler.GetPoll)
			r.With(AuthMiddleware).Post("/{id}/votes", voteHandler.VoteOnPoll)
		})
	})

	r.Route("/oauth", func(r chi.Router) {
		r.Post("/callback", authHandler.GoogleCallback)
	})

	r.Route("/auth", func(r chi.Router) {
		r.Post("/logout", authHandler.Logout)
		r.Post("/refresh", authHandler.Refresh)
	})

	return r
}
