package assistant

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

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
	mux.HandleFunc("GET /api/v1/documents/{id}/assist", h.getAssist)
	mux.HandleFunc("POST /api/v1/documents/{id}/assist/analyze", h.analyze)
	mux.HandleFunc("POST /api/v1/assistant/chat", h.chat)
}

func (h *Handler) getAssist(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	st, err := h.svc.GetStatus(r.Context(), httpx.OwnerFrom(r.Context()), id)
	if writeAssistErr(w, err) {
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}

func (h *Handler) analyze(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	if !h.svc.Available() {
		httpx.Error(w, http.StatusServiceUnavailable, "assistant_unavailable", "assistente IA não configurado (GROQ_API_KEY)")
		return
	}
	st, err := h.svc.Analyze(r.Context(), httpx.OwnerFrom(r.Context()), id)
	if writeAssistErr(w, err) {
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}

func (h *Handler) chat(w http.ResponseWriter, r *http.Request) {
	if !h.svc.Available() {
		httpx.Error(w, http.StatusServiceUnavailable, "assistant_unavailable", "assistente IA não configurado (GROQ_API_KEY)")
		return
	}
	var body ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "JSON inválido")
		return
	}
	wantStream := r.URL.Query().Get("stream") == "1" ||
		strings.Contains(r.Header.Get("Accept"), "text/event-stream")
	if !wantStream {
		resp, err := h.svc.Chat(r.Context(), httpx.OwnerFrom(r.Context()), body)
		if writeAssistErr(w, err) {
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.Error(w, http.StatusInternalServerError, "internal", "streaming não suportado")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	writeEvent := func(v any) error {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	resp, err := h.svc.ChatStream(r.Context(), httpx.OwnerFrom(r.Context()), body, func(delta string) error {
		return writeEvent(map[string]string{"delta": delta})
	})
	if err != nil {
		_ = writeEvent(map[string]string{"error": err.Error()})
		return
	}
	_ = writeEvent(map[string]any{
		"done":       true,
		"citations":  resp.Citations,
		"disclaimer": resp.Disclaimer,
	})
}

func writeAssistErr(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "forbidden", "acesso negado a documento de outro titular")
	case errors.Is(err, ErrDocumentGone), errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "documento ou análise não encontrada")
	case errors.Is(err, ErrUnavailable):
		httpx.Error(w, http.StatusServiceUnavailable, "assistant_unavailable", "assistente IA não configurado")
	case errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusBadRequest, "invalid_input", err.Error())
	default:
		slog.Error("assistant error", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha no assistente")
	}
	return true
}
