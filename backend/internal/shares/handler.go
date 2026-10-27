package shares

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/google/uuid"

	"carteira/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/documents/{id}/shares", h.create)
	mux.HandleFunc("GET /api/v1/documents/{id}/shares", h.list)
	mux.HandleFunc("POST /api/v1/shares/{id}/revoke", h.revoke)
	mux.HandleFunc("GET /api/v1/public/shares/{token}", h.publicGet)
	mux.HandleFunc("GET /api/v1/public/shares/{token}/file", h.publicFile)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	var body struct {
		Days int `json:"days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "envie JSON com days")
		return
	}
	sh, err := h.svc.Create(r.Context(), httpx.OwnerFrom(r.Context()), id, body.Days)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusCreated, sh)
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "documento não encontrado")
	case errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusUnprocessableEntity, "invalid_metadata", err.Error())
	default:
		slog.Error("create share failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao gerar link")
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	items, err := h.svc.List(r.Context(), httpx.OwnerFrom(r.Context()), id)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "documento não encontrado")
	default:
		slog.Error("list shares failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao listar links")
	}
}

func (h *Handler) revoke(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	sh, err := h.svc.Revoke(r.Context(), httpx.OwnerFrom(r.Context()), id)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, sh)
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "link não encontrado")
	default:
		slog.Error("revoke share failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao revogar link")
	}
}

func (h *Handler) publicGet(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		httpx.Error(w, http.StatusBadRequest, "invalid_token", "token inválido")
		return
	}
	pub, err := h.svc.PublicGet(r.Context(), token)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, pub)
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrDocumentGone):
		httpx.Error(w, http.StatusNotFound, "not_found", "link inválido")
	case errors.Is(err, ErrGone):
		httpx.Error(w, http.StatusGone, "gone", "link expirado ou revogado")
	default:
		slog.Error("public share failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao validar link")
	}
}

func (h *Handler) publicFile(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		httpx.Error(w, http.StatusBadRequest, "invalid_token", "token inválido")
		return
	}
	payload, err := h.svc.PublicFile(r.Context(), token)
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrDocumentGone):
		httpx.Error(w, http.StatusNotFound, "not_found", "link inválido")
		return
	case errors.Is(err, ErrGone):
		httpx.Error(w, http.StatusGone, "gone", "link expirado ou revogado")
		return
	case err != nil:
		slog.Error("public share file failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao obter arquivo")
		return
	}
	defer payload.Reader.Close()

	disposition := "inline"
	if r.URL.Query().Get("download") == "1" {
		disposition = "attachment"
	}
	w.Header().Set("Content-Type", payload.ContentType)
	w.Header().Set("Content-Disposition", disposition+"; filename*=UTF-8''"+url.PathEscape(payload.FileName))
	if payload.Size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(payload.Size, 10))
	}
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, payload.Reader); err != nil {
		slog.Error("public file stream interrupted", "err", err)
	}
}
