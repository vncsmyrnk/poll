package http

import (
	"net/http"

	"github.com/poll/api/internal/core/ports"
)

type AuthHandler struct {
	authService ports.AuthService
	redirectURL string
}

func NewAuthHandler(authService ports.AuthService, redirectURL string) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		redirectURL: redirectURL,
	}
}

type googleCallbackRequest struct {
	Credential string `json:"credential"`
}

func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	credential := r.FormValue("credential")
	if credential == "" {
		http.Error(w, "Missing credential", http.StatusBadRequest)
		return
	}

	accessToken, refreshToken, err := h.authService.LoginWithGoogle(r.Context(), credential)
	if err != nil {
		http.Error(w, "Authentication failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	h.setAccessTokenCookie(w, accessToken)
	h.setRefreshTokenCookie(w, refreshToken)

	http.Redirect(w, r, h.redirectURL, http.StatusSeeOther)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "Missing refresh token", http.StatusUnauthorized)
		return
	}

	accessToken, refreshToken, err := h.authService.RefreshAccessToken(r.Context(), cookie.Value)
	if err != nil {
		h.expireCookies(w)
		http.Error(w, "Refresh failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	h.setAccessTokenCookie(w, accessToken)

	// If refresh token was rotated, update it too
	if refreshToken != "" && refreshToken != cookie.Value {
		h.setRefreshTokenCookie(w, refreshToken)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *AuthHandler) setAccessTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // Should be true in production
		SameSite: http.SameSiteLaxMode,
		MaxAge:   15 * 60, // 15 minutes
	})
}

func (h *AuthHandler) setRefreshTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})
}

func (h *AuthHandler) expireCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "access_token", MaxAge: -1, Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", MaxAge: -1, Path: "/"})
}
