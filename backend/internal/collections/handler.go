package collections

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

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
	mux.HandleFunc("GET /api/v1/collections", h.list)
	mux.HandleFunc("POST /api/v1/collections", h.create)
	mux.HandleFunc("GET /api/v1/collections/{id}", h.get)
	mux.HandleFunc("PATCH /api/v1/collections/{id}", h.patch)
	mux.HandleFunc("DELETE /api/v1/collections/{id}", h.delete)
	mux.HandleFunc("POST /api/v1/collections/{id}/documents", h.addDocuments)
	mux.HandleFunc("DELETE /api/v1/collections/{id}/documents/{docId}", h.removeDocument)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List(r.Context(), httpx.OwnerFrom(r.Context()))
	if err != nil {
		slog.Error("list collections failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao listar coleções")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "envie JSON com name")
		return
	}
	c, err := h.svc.Create(r.Context(), httpx.OwnerFrom(r.Context()), body.Name)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusCreated, c)
	case errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusUnprocessableEntity, "invalid_metadata", err.Error())
	default:
		slog.Error("create collection failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao criar coleção")
	}
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	d, err := h.svc.Get(r.Context(), httpx.OwnerFrom(r.Context()), id)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, d)
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "coleção não encontrada")
	default:
		slog.Error("get collection failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao obter coleção")
	}
}

func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "envie JSON com name")
		return
	}
	c, err := h.svc.Rename(r.Context(), httpx.OwnerFrom(r.Context()), id, body.Name)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, c)
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "coleção não encontrada")
	case errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusUnprocessableEntity, "invalid_metadata", err.Error())
	default:
		slog.Error("rename collection failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao renomear coleção")
	}
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	err = h.svc.Delete(r.Context(), httpx.OwnerFrom(r.Context()), id)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "coleção não encontrada")
	default:
		slog.Error("delete collection failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao excluir coleção")
	}
}

func (h *Handler) addDocuments(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	var body struct {
		DocumentIDs []uuid.UUID `json:"document_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "envie JSON com document_ids")
		return
	}
	d, err := h.svc.AddDocuments(r.Context(), httpx.OwnerFrom(r.Context()), id, body.DocumentIDs)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, d)
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "coleção ou documento não encontrado")
	case errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusUnprocessableEntity, "invalid_metadata", err.Error())
	default:
		slog.Error("add collection documents failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao adicionar documentos")
	}
}

func (h *Handler) removeDocument(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	docID, err := uuid.Parse(r.PathValue("docId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "document_id inválido")
		return
	}
	err = h.svc.RemoveDocument(r.Context(), httpx.OwnerFrom(r.Context()), id, docID)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "coleção ou vínculo não encontrado")
	default:
		slog.Error("remove collection document failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao remover documento")
	}
}
