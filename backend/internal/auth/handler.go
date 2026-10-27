package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"carteira/internal/httpx"
)

type Handler struct {
	svc    *Service
	secure bool
}

func NewHandler(svc *Service, secureCookie bool) *Handler {
	return &Handler{svc: svc, secure: secureCookie}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/register", h.register)
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("POST /api/v1/auth/logout", h.logout)
	mux.HandleFunc("GET /api/v1/auth/me", h.me)
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "envie JSON com email e password")
		return
	}
	u, token, err := h.svc.Register(r.Context(), body.Email, body.Password, body.Name)
	switch {
	case err == nil:
		h.setSessionCookie(w, token)
		httpx.JSON(w, http.StatusCreated, u)
	case errors.Is(err, ErrConflict):
		httpx.Error(w, http.StatusConflict, "email_taken", "e-mail já cadastrado")
	case errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusUnprocessableEntity, "invalid_metadata", err.Error())
	default:
		slog.Error("register failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao cadastrar")
	}
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "envie JSON com email e password")
		return
	}
	u, token, err := h.svc.Login(r.Context(), body.Email, body.Password)
	switch {
	case err == nil:
		h.setSessionCookie(w, token)
		httpx.JSON(w, http.StatusOK, u)
	case errors.Is(err, ErrInvalidCreds), errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusUnauthorized, "invalid_credentials", "e-mail ou senha inválidos")
	default:
		slog.Error("login failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao autenticar")
	}
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	_ = h.svc.Logout(r.Context(), cookieToken(r))
	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	u, err := h.svc.UserFromToken(r.Context(), cookieToken(r))
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, u)
	case errors.Is(err, ErrUnauthorized):
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "sessão inválida ou expirada")
	default:
		slog.Error("me failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao obter titular")
	}
}

// Middleware validates the session cookie and sets owner context. Public paths skip auth.
func (h *Handler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicAPI(r.Method, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		u, err := h.svc.UserFromToken(r.Context(), cookieToken(r))
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "unauthorized", "autenticação necessária")
			return
		}
		ctx := httpx.WithOwner(r.Context(), u.OwnerID())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isPublicAPI(method, path string) bool {
	if path == "/api/v1/health" {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/public/") {
		return true
	}
	if method == http.MethodPost &&
		(path == "/api/v1/auth/register" || path == "/api/v1/auth/login" || path == "/api/v1/auth/logout") {
		return true
	}
	return false
}

func cookieToken(r *http.Request) string {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.secure,
		Expires:  time.Now().UTC().Add(SessionTTL),
		MaxAge:   int(SessionTTL.Seconds()),
	})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.secure,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
	})
}
