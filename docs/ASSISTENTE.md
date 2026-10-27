# Assistente (OCR + IA + RAG)

**Escopo:** pós-MVP assistido. **Não** altera RF-23: autenticidade do emissor permanece **não verificada**.

## O que faz

1. **OCR** do arquivo: PDF combina camada de texto (`pdftotext`) **e** Tesseract nas páginas rasterizadas (CNH digital Serpro costuma ter só a capa na text layer); imagens com Tesseract (`por+eng`); fallback opcional de visão Groq.
2. **Coerência assistida** — LLM Groq compara texto OCR com metadados (`title`, `type`, `origin`, tags). Resultados: `consistent` | `divergent` | `unreadable` | `pending` | `error`.
3. **RAG lexical** — chunks em Postgres (`tsvector`) com knowledge do produto + texto OCR do titular.
4. **FAB global** — chat nas rotas autenticadas (`Layout`), com contexto opcional do documento da URL.

**Deps de host:** `tesseract-ocr`, `tesseract-ocr-por`, `poppler-utils` (`pdftotext`/`pdftoppm`).

## O que NÃO faz

- Não prova autenticidade de órgão emissor / ICP-Brasil / Gov.br.
- Não usa DocTR nem sidecar Python.
- Não persiste histórico de chat; não aparece em `/v` ou `/i`.

## Env

| Variável | Uso |
|---|---|
| `GROQ_API_KEY` | Obrigatória para habilitar o assistente |
| `GROQ_API_MODEL` | Chat (default `openai/gpt-oss-120b`) |
| `GROQ_VISION_MODEL` | Fallback de visão (opcional; OCR principal é Tesseract) |

Sem key → API responde `503 assistant_unavailable`; upload/integridade seguem normais.

## API

| Método | Rota | Auth |
|---|---|---|
| `GET` | `/api/v1/documents/{id}/assist` | sim |
| `POST` | `/api/v1/documents/{id}/assist/analyze` | sim |
| `POST` | `/api/v1/assistant/chat` | sim — body `{ messages, document_id? }` |

Após `POST /documents` ou nova versão, a análise é **enfileirada** em background (não bloqueia o hash).

## Pacote

`backend/internal/assistant/` — Groq client, OCR, FTS, coerência, handlers.  
UI: `AssistantFab.tsx`, painel no `DocumentDetail.tsx`.

## Tabelas (migration 0009)

- `document_extractions`
- `consistency_checks`
- `assistant_chunks` (seed knowledge + documentos)
