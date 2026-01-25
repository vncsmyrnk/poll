package http

import (
	"net/http"

	"github.com/vncsmyrnk/poll/internal/core/ports"
)

type AuthHandler struct {
	authService    ports.AuthService
	redirectURL    string
	cookieDomain   string
	cookieSameSite http.SameSite
}

func NewAuthHandler(authService ports.AuthService, redirectURL string, cookieDomain string, cookieSameSite http.SameSite) *AuthHandler {
	return &AuthHandler{
		authService:    authService,
		redirectURL:    redirectURL,
		cookieDomain:   cookieDomain,
		cookieSameSite: cookieSameSite,
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

// Refresh godoc
// @Summary      Refreshes the autheticated user out
// @Description  Creates a new access token cookie based on the refresh token. This cookie is used as authentication for `/api` calls.
// @Tags         auth
// @Accept       json
// @Success      200
// @Failure      401
// @Router       /auth/refresh [post]
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
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

// Logout godoc
// @Summary      Logs the autheticated user out
// @Description  Clears the refresh token cookie
// @Tags         auth
// @Accept       json
// @Success      200
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie.Value != "" {
		_ = h.authService.Logout(r.Context(), cookie.Value)
	}

	h.expireCookies(w)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *AuthHandler) setAccessTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    token,
		Path:     "/",
		Domain:   h.cookieDomain,
		HttpOnly: true,
		Secure:   true,
		SameSite: h.cookieSameSite,
		MaxAge:   15 * 60, // 15 minutes
	})
}

func (h *AuthHandler) setRefreshTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/",
		Domain:   h.cookieDomain,
		HttpOnly: true,
		Secure:   true,
		SameSite: h.cookieSameSite,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})
}

func (h *AuthHandler) expireCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "access_token", MaxAge: -1, Path: "/", Domain: h.cookieDomain})
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", MaxAge: -1, Path: "/", Domain: h.cookieDomain})
}
