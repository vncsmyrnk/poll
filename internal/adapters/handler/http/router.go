package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "github.com/vncsmyrnk/poll/docs"
)

// @title           Poll API
// @version         1.0
// @description     Simple open source polling REST API.

// @host      https://poll-api.vncsmyrnk.dev
// @BasePath  /api

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
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
			r.With(AuthMiddleware).Get("/{id}/count", pollHandler.GetPollStats)

			r.Route("/{id}/votes", func(r chi.Router) {
				r.Use(AuthMiddleware)
				r.Post("/", voteHandler.VoteOnPoll)
			})
			r.With(AuthMiddleware).Get("/{id}/my-vote", voteHandler.GetMyVote)
		})
	})

	r.Route("/oauth", func(r chi.Router) {
		r.Post("/callback", authHandler.GoogleCallback)
	})

	r.Route("/auth", func(r chi.Router) {
		r.Post("/logout", authHandler.Logout)
		r.Post("/refresh", authHandler.Refresh)
	})

	r.Get("/swagger/*", httpSwagger.Handler())

	return r
}
