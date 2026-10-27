package documents

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
	mux.HandleFunc("POST /api/v1/documents", h.upload)
	mux.HandleFunc("GET /api/v1/documents", h.list)
	mux.HandleFunc("GET /api/v1/documents/{id}", h.get)
	mux.HandleFunc("PATCH /api/v1/documents/{id}", h.patch)
	mux.HandleFunc("DELETE /api/v1/documents/{id}", h.delete)
	mux.HandleFunc("GET /api/v1/documents/{id}/file", h.file)
	mux.HandleFunc("GET /api/v1/documents/{id}/receipt", h.receipt)
	mux.HandleFunc("POST /api/v1/documents/{id}/verify", h.verify)
	mux.HandleFunc("GET /api/v1/documents/{id}/verifications", h.verifications)
	mux.HandleFunc("POST /api/v1/documents/{id}/revoke", h.revoke)
	mux.HandleFunc("POST /api/v1/documents/{id}/versions", h.addVersion)
	mux.HandleFunc("GET /api/v1/documents/{id}/versions", h.listVersions)
	mux.HandleFunc("GET /api/v1/documents/{id}/integrity", h.integrity)
	mux.HandleFunc("GET /api/v1/document-types", h.types)
	mux.HandleFunc("GET /api/v1/audit", h.audit)
	mux.HandleFunc("GET /api/v1/public/documents/{id}/integrity", h.publicIntegrity)
	mux.HandleFunc("GET /api/v1/public/documents/{id}/file", h.publicFile)
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	// 1 MiB of headroom for the other multipart fields.
	r.Body = http.MaxBytesReader(w, r.Body, h.svc.MaxBytes()+1<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			httpx.Error(w, http.StatusRequestEntityTooLarge, "file_too_large", "arquivo excede o limite permitido")
			return
		}
		httpx.Error(w, http.StatusBadRequest, "invalid_form", "envie multipart/form-data com o campo file")
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "missing_file", "campo file é obrigatório")
		return
	}
	defer file.Close()

	expiresAt, err := ParseExpiresAt(r.FormValue("expires_at"))
	if err != nil {
		httpx.Error(w, http.StatusUnprocessableEntity, "invalid_metadata", err.Error())
		return
	}

	res, err := h.svc.Upload(r.Context(), UploadInput{
		OwnerID:   httpx.OwnerFrom(r.Context()),
		Title:     r.FormValue("title"),
		Type:      r.FormValue("type"),
		Origin:    r.FormValue("origin"),
		Tags:      r.MultipartForm.Value["tags"],
		FileName:  header.Filename,
		Size:      header.Size,
		ExpiresAt: expiresAt,
	}, file)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusCreated, res)
	case errors.Is(err, ErrTooLarge):
		httpx.Error(w, http.StatusRequestEntityTooLarge, "file_too_large", "arquivo excede o limite permitido")
	case errors.Is(err, ErrUnsupportedType):
		httpx.Error(w, http.StatusUnsupportedMediaType, "unsupported_type", "apenas PDF, JPEG e PNG são aceitos")
	case errors.Is(err, ErrInvalidMetadata):
		httpx.Error(w, http.StatusUnprocessableEntity, "invalid_metadata", err.Error())
	default:
		slog.Error("upload failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao salvar o documento")
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := h.svc.List(r.Context(), httpx.OwnerFrom(r.Context()), ListFilter{
		Q: q.Get("q"), Type: q.Get("type"), Status: q.Get("status"), Tag: q.Get("tag"),
		Collection: q.Get("collection"),
	})
	if err != nil {
		slog.Error("list failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao listar documentos")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	detail, err := h.svc.Get(r.Context(), httpx.OwnerFrom(r.Context()), id)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, detail)
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "forbidden", "acesso negado a documento de outro titular")
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "documento não encontrado")
	default:
		slog.Error("get document failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao consultar documento")
	}
}

func (h *Handler) file(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	payload, err := h.svc.File(r.Context(), httpx.OwnerFrom(r.Context()), id)
	switch {
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "forbidden", "acesso negado a documento de outro titular")
		return
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "forbidden", "acesso negado a documento de outro titular")
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "documento não encontrado")
		return
	case err != nil:
		slog.Error("file download failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao baixar arquivo")
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
		slog.Error("file stream interrupted", "err", err)
	}
}

func (h *Handler) publicIntegrity(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	pub, err := h.svc.PublicIntegrity(r.Context(), id)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, pub)
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "documento não encontrado")
	default:
		slog.Error("public integrity failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao verificar integridade")
	}
}

func (h *Handler) publicFile(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	payload, err := h.svc.PublicFile(r.Context(), id)
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "documento não encontrado")
		return
	case err != nil:
		slog.Error("public document file failed", "err", err)
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
		slog.Error("public document file stream interrupted", "err", err)
	}
}

func (h *Handler) receipt(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	pdf, filename, err := h.svc.ReceiptPDF(r.Context(), httpx.OwnerFrom(r.Context()), id)
	switch {
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "forbidden", "acesso negado a documento de outro titular")
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "documento não encontrado")
	case errors.Is(err, ErrMissingIntegrity):
		httpx.Error(w, http.StatusUnprocessableEntity, "missing_integrity", "documento sem registro de integridade")
	case err != nil:
		slog.Error("receipt pdf failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao gerar recibo")
	default:
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
		w.Header().Set("Content-Length", strconv.Itoa(len(pdf)))
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(pdf); err != nil {
			slog.Error("receipt stream interrupted", "err", err)
		}
	}
}

func (h *Handler) addVersion(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, h.svc.MaxBytes()+1<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			httpx.Error(w, http.StatusRequestEntityTooLarge, "file_too_large", "arquivo excede o limite permitido")
			return
		}
		httpx.Error(w, http.StatusBadRequest, "invalid_form", "envie multipart/form-data com o campo file")
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "missing_file", "campo file é obrigatório")
		return
	}
	defer file.Close()

	res, err := h.svc.AddVersion(r.Context(), httpx.OwnerFrom(r.Context()), id, header.Filename, header.Size, file)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusCreated, res)
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "forbidden", "acesso negado a documento de outro titular")
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "documento não encontrado")
	case errors.Is(err, ErrTooLarge):
		httpx.Error(w, http.StatusRequestEntityTooLarge, "file_too_large", "arquivo excede o limite permitido")
	case errors.Is(err, ErrUnsupportedType):
		httpx.Error(w, http.StatusUnsupportedMediaType, "unsupported_type", "apenas PDF, JPEG e PNG são aceitos")
	case errors.Is(err, ErrInvalidMetadata):
		httpx.Error(w, http.StatusUnprocessableEntity, "invalid_metadata", err.Error())
	case errors.Is(err, ErrInvalidTransition):
		httpx.Error(w, http.StatusConflict, "invalid_transition", "só documentos ativos podem ser substituídos")
	default:
		slog.Error("add version failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao substituir documento")
	}
}

func (h *Handler) listVersions(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	items, err := h.svc.ListVersions(r.Context(), httpx.OwnerFrom(r.Context()), id)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "forbidden", "acesso negado a documento de outro titular")
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "documento não encontrado")
	default:
		slog.Error("list versions failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao listar versões")
	}
}

func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	var body MetadataPatch
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "envie JSON com title, type, origin, tags, expires_at")
		return
	}
	doc, err := h.svc.UpdateMetadata(r.Context(), httpx.OwnerFrom(r.Context()), id, body)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, doc)
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "forbidden", "acesso negado a documento de outro titular")
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "documento não encontrado")
	case errors.Is(err, ErrInvalidMetadata):
		httpx.Error(w, http.StatusUnprocessableEntity, "invalid_metadata", err.Error())
	case errors.Is(err, ErrInvalidTransition):
		httpx.Error(w, http.StatusConflict, "invalid_transition", "metadados não podem ser alterados neste estado")
	default:
		slog.Error("patch metadata failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao atualizar metadados")
	}
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	err = h.svc.SoftDelete(r.Context(), httpx.OwnerFrom(r.Context()), id)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "forbidden", "acesso negado a documento de outro titular")
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "documento não encontrado")
	default:
		slog.Error("soft delete failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao excluir documento")
	}
}

func (h *Handler) revoke(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "envie JSON com o campo reason")
		return
	}
	doc, err := h.svc.Revoke(r.Context(), httpx.OwnerFrom(r.Context()), id, body.Reason)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, doc)
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "forbidden", "acesso negado a documento de outro titular")
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "documento não encontrado")
	case errors.Is(err, ErrInvalidMetadata):
		httpx.Error(w, http.StatusUnprocessableEntity, "invalid_metadata", "motivo da revogação é obrigatório")
	case errors.Is(err, ErrInvalidTransition):
		httpx.Error(w, http.StatusConflict, "invalid_transition", "documento não pode ser revogado neste estado")
	default:
		slog.Error("revoke failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao revogar documento")
	}
}

func (h *Handler) verify(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	rec, err := h.svc.Verify(r.Context(), httpx.OwnerFrom(r.Context()), id)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, rec)
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "forbidden", "acesso negado a documento de outro titular")
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "documento não encontrado")
	default:
		slog.Error("verify failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao verificar integridade")
	}
}

func (h *Handler) verifications(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	items, err := h.svc.ListVerifications(r.Context(), httpx.OwnerFrom(r.Context()), id)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "forbidden", "acesso negado a documento de outro titular")
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "documento não encontrado")
	default:
		slog.Error("list verifications failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao listar verificações")
	}
}

func (h *Handler) integrity(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id inválido")
		return
	}
	rec, err := h.svc.Integrity(r.Context(), httpx.OwnerFrom(r.Context()), id)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, rec)
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "forbidden", "acesso negado a documento de outro titular")
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "documento não encontrado")
	default:
		slog.Error("integrity lookup failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao consultar integridade")
	}
}

func (h *Handler) types(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{"types": Types, "mime_types": AllowedMIME})
}

func (h *Handler) audit(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListAudit(r.Context(), httpx.OwnerFrom(r.Context()))
	if err != nil {
		slog.Error("list audit failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "falha ao listar auditoria")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}
