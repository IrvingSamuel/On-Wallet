package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ObjectStore interface {
	Get(ctx context.Context, key string) (io.ReadCloser, int64, string, error)
}

type Service struct {
	repo  *Repository
	store ObjectStore
	groq  *GroqClient

	mu    sync.Mutex
	inflight map[string]bool
}

func NewService(repo *Repository, store ObjectStore, groq *GroqClient) *Service {
	return &Service{
		repo:     repo,
		store:    store,
		groq:     groq,
		inflight: map[string]bool{},
	}
}

func (s *Service) Available() bool {
	return s.groq != nil && s.groq.Enabled()
}

// EnqueueAnalyze runs OCR+consistency in background after upload/version.
func (s *Service) EnqueueAnalyze(ownerID string, docID uuid.UUID) {
	if !s.Available() {
		return
	}
	key := ownerID + ":" + docID.String()
	s.mu.Lock()
	if s.inflight[key] {
		s.mu.Unlock()
		return
	}
	s.inflight[key] = true
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.inflight, key)
			s.mu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if _, err := s.Analyze(ctx, ownerID, docID); err != nil {
			slog.Warn("assist analyze failed", "doc", docID, "err", err)
		}
	}()
}

func (s *Service) GetStatus(ctx context.Context, ownerID string, docID uuid.UUID) (AssistStatus, error) {
	if _, err := s.repo.GetOwnedLatest(ctx, ownerID, docID); err != nil {
		return AssistStatus{}, err
	}
	out := AssistStatus{Disclaimer: Disclaimer}
	if ex, err := s.repo.LatestExtraction(ctx, ownerID, docID); err == nil {
		// Don't leak huge OCR text on every poll unless done — trim for API.
		if len(ex.RawText) > 4000 {
			ex.RawText = ex.RawText[:4000] + "…"
		}
		out.Extraction = ex
	} else if err != ErrNotFound {
		return AssistStatus{}, err
	}
	if c, err := s.repo.LatestConsistency(ctx, ownerID, docID); err == nil {
		out.Consistency = c
	} else if err != ErrNotFound {
		return AssistStatus{}, err
	}
	return out, nil
}

func (s *Service) Analyze(ctx context.Context, ownerID string, docID uuid.UUID) (AssistStatus, error) {
	if !s.Available() {
		return AssistStatus{}, ErrUnavailable
	}
	meta, err := s.repo.GetOwnedLatest(ctx, ownerID, docID)
	if err != nil {
		return AssistStatus{}, err
	}

	ex := &Extraction{
		ID:         uuid.New(),
		DocumentID: meta.DocumentID,
		VersionID:  meta.VersionID,
		OwnerID:    ownerID,
		Status:     ExtractRunning,
		Fields:     json.RawMessage(`{}`),
	}
	if err := s.repo.UpsertExtractionRunning(ctx, ex); err != nil {
		return AssistStatus{}, err
	}

	rc, _, _, err := s.store.Get(ctx, meta.ObjectKey)
	if err != nil {
		ex.Status = ExtractError
		ex.Error = "falha ao ler arquivo armazenado"
		_ = s.repo.UpsertExtractionRunning(ctx, ex)
		return s.finishError(ctx, meta, ownerID, "Não foi possível ler o arquivo para OCR.")
	}
	data, err := io.ReadAll(io.LimitReader(rc, 25<<20))
	rc.Close()
	if err != nil {
		ex.Status = ExtractError
		ex.Error = "falha ao carregar bytes"
		_ = s.repo.UpsertExtractionRunning(ctx, ex)
		return s.finishError(ctx, meta, ownerID, "Falha ao carregar o arquivo para análise.")
	}

	ocr, err := s.extractText(ctx, meta.MimeType, data)
	if err != nil {
		ex.Status = ExtractError
		ex.Error = err.Error()
		_ = s.repo.UpsertExtractionRunning(ctx, ex)
		return s.finishError(ctx, meta, ownerID, "Falha no OCR/visão: "+err.Error())
	}

	if ocr.Method == "unreadable" || strings.TrimSpace(ocr.Text) == "" {
		ex.Status = ExtractUnreadable
		ex.RawText = ocr.Text
		ex.Error = "texto insuficiente"
		_ = s.repo.UpsertExtractionRunning(ctx, ex)
		check := ConsistencyCheck{
			ID: uuid.New(), DocumentID: meta.DocumentID, VersionID: meta.VersionID, OwnerID: ownerID,
			Result: ResultUnreadable, Summary: "OCR insuficiente para coerência assistida.",
			Details: json.RawMessage(`{"method":"unreadable"}`),
		}
		_ = s.repo.InsertConsistency(ctx, &check)
		return s.GetStatus(ctx, ownerID, docID)
	}

	check, fields, err := s.analyzeConsistency(ctx, meta, ocr.Text)
	if err != nil {
		ex.Status = ExtractError
		ex.Error = err.Error()
		ex.RawText = ocr.Text
		_ = s.repo.UpsertExtractionRunning(ctx, ex)
		return s.finishError(ctx, meta, ownerID, "Falha na análise de coerência.")
	}

	ex.Status = ExtractDone
	ex.RawText = ocr.Text
	ex.Fields = fields
	ex.Error = ""
	if err := s.repo.UpsertExtractionRunning(ctx, ex); err != nil {
		return AssistStatus{}, err
	}
	_ = s.repo.ReplaceDocumentChunks(ctx, ownerID, meta.DocumentID, meta.VersionID,
		fmt.Sprintf("%s (v%d)", meta.Title, meta.Version), ocr.Text)

	check.ID = uuid.New()
	check.DocumentID = meta.DocumentID
	check.VersionID = meta.VersionID
	check.OwnerID = ownerID
	if err := s.repo.InsertConsistency(ctx, &check); err != nil {
		return AssistStatus{}, err
	}
	return s.GetStatus(ctx, ownerID, docID)
}

func (s *Service) finishError(ctx context.Context, meta docVersionRow, ownerID, summary string) (AssistStatus, error) {
	details, _ := json.Marshal(map[string]string{"stage": "analyze"})
	_ = s.repo.InsertConsistency(ctx, &ConsistencyCheck{
		ID: uuid.New(), DocumentID: meta.DocumentID, VersionID: meta.VersionID, OwnerID: ownerID,
		Result: ResultError, Summary: summary, Details: details,
	})
	return s.GetStatus(ctx, ownerID, meta.DocumentID)
}

const chatSystem = `Você é o Wall-et bot, assistente do On-Wallet / Carteira de Documentos (Project LAB · UNIMA Afya). Responda em português, de forma objetiva.
Formato: use Markdown leve (listas numeradas, **negrito**, blocos curtos) para facilitar a leitura.
Regras obrigatórias:
- Integridade (SHA-256) ≠ autenticidade do emissor.
- Coerência assistida (OCR/IA) NÃO prova autenticidade; nunca diga que o documento é autêntico ou fraudulento com certeza.
- Use o contexto RAG quando relevante; se não souber, diga.
- Ajude com uso da carteira (upload, verificação, recibo, compartilhamento, coleções).`

type chatPrep struct {
	system    string
	messages  []ChatMessage
	citations []string
}

func (s *Service) prepareChat(ctx context.Context, ownerID string, req ChatRequest) (chatPrep, error) {
	if !s.Available() {
		return chatPrep{}, ErrUnavailable
	}
	if len(req.Messages) == 0 {
		return chatPrep{}, fmt.Errorf("%w: messages required", ErrInvalidInput)
	}
	lastUser := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" && strings.TrimSpace(req.Messages[i].Content) != "" {
			lastUser = req.Messages[i].Content
			break
		}
	}
	if lastUser == "" {
		return chatPrep{}, fmt.Errorf("%w: user message required", ErrInvalidInput)
	}

	query := lastUser
	var docContext string
	if req.DocumentID != nil && *req.DocumentID != "" {
		docID, err := uuid.Parse(*req.DocumentID)
		if err != nil {
			return chatPrep{}, fmt.Errorf("%w: document_id", ErrInvalidInput)
		}
		st, err := s.GetStatus(ctx, ownerID, docID)
		if err != nil {
			return chatPrep{}, err
		}
		query = lastUser + " documento"
		if st.Extraction != nil && st.Extraction.RawText != "" {
			docContext = "\n\nTexto OCR do documento em foco (trecho):\n" + truncate(st.Extraction.RawText, 3000)
		}
		if st.Consistency != nil {
			docContext += fmt.Sprintf("\nCoerência assistida: %s — %s", st.Consistency.Result, st.Consistency.Summary)
		}
		meta, err := s.repo.GetOwnedLatest(ctx, ownerID, docID)
		if err == nil {
			docContext += fmt.Sprintf("\nMetadados: title=%s type=%s origin=%s", meta.Title, meta.Type, meta.Origin)
		}
	}

	chunks, err := s.repo.SearchChunks(ctx, ownerID, query, 6)
	if err != nil {
		return chatPrep{}, err
	}
	var citations []string
	var kb strings.Builder
	kb.WriteString("Contexto recuperado (RAG lexical):\n")
	for _, c := range chunks {
		citations = append(citations, c.Title)
		kb.WriteString("- ")
		kb.WriteString(c.Title)
		kb.WriteString(": ")
		kb.WriteString(truncate(c.Body, 800))
		kb.WriteByte('\n')
	}
	kb.WriteString(docContext)

	msgs := append([]ChatMessage{}, req.Messages...)
	msgs = append([]ChatMessage{{Role: "user", Content: kb.String() + "\n(Use o contexto acima só se ajudar; a pergunta do titular segue nas mensagens.)"}}, msgs...)
	return chatPrep{system: chatSystem, messages: msgs, citations: citations}, nil
}

func (s *Service) Chat(ctx context.Context, ownerID string, req ChatRequest) (ChatResponse, error) {
	prep, err := s.prepareChat(ctx, ownerID, req)
	if err != nil {
		return ChatResponse{}, err
	}
	reply, err := s.groq.Chat(ctx, prep.system, prep.messages)
	if err != nil {
		return ChatResponse{}, err
	}
	return ChatResponse{Reply: reply, Citations: prep.citations, Disclaimer: Disclaimer}, nil
}

// ChatStream streams reply deltas; caller receives citations after the stream via return.
func (s *Service) ChatStream(ctx context.Context, ownerID string, req ChatRequest, onDelta func(string) error) (ChatResponse, error) {
	prep, err := s.prepareChat(ctx, ownerID, req)
	if err != nil {
		return ChatResponse{}, err
	}
	var full strings.Builder
	err = s.groq.ChatStream(ctx, prep.system, prep.messages, func(delta string) error {
		full.WriteString(delta)
		return onDelta(delta)
	})
	if err != nil {
		return ChatResponse{}, err
	}
	return ChatResponse{Reply: strings.TrimSpace(full.String()), Citations: prep.citations, Disclaimer: Disclaimer}, nil
}
