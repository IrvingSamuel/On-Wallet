# API v1

Base: `https://carteira.eflow.space/api/v1`

Erros sempre seguem o formato `{"error": {"code": "...", "message": "..."}}`.

Autenticação (RF-01/02): cookie HttpOnly `carteira_session` após `POST /auth/register` ou `POST /auth/login`. Rotas privadas exigem sessão válida (`401` sem cookie). Isolamento entre titulares: documento de outro dono → `403 forbidden`. Público sem login: `GET /health`, `GET /public/shares/{token}`, `GET /public/shares/{token}/file`, `GET /public/documents/{id}/integrity`, `GET /public/documents/{id}/file`, register/login/logout. Assistente (OCR/IA/RAG): rotas `/documents/{id}/assist*` e `/assistant/chat` — ver [ASSISTENTE.md](./ASSISTENTE.md).

## POST /auth/register

Cadastro do titular. Body `{ "email", "password", "name?" }`. Senha ≥ 8 caracteres. Resposta `201` + cookie de sessão.

```bash
curl -c jar.txt -X POST -H 'Content-Type: application/json' \
  -d '{"email":"a@b.com","password":"senha-segura","name":"Ana"}' \
  https://carteira.eflow.space/api/v1/auth/register
```

## POST /auth/login

Body `{ "email", "password" }`. Resposta `200` + cookie. Credenciais inválidas → `401`.

## POST /auth/logout

Encerra a sessão (`204`) e limpa o cookie.

## GET /auth/me

Titular da sessão atual. Sem sessão → `401`.

## GET /health

Verifica a API, o Postgres e o MinIO. Responde `200` se tudo está ok e `503` se algo falhou.

```bash
curl https://carteira.eflow.space/api/v1/health
# {"checks":{"minio":"ok","postgres":"ok"},"status":"ok"}
```

## GET /document-types

Lista os tipos aceitos em `type` e os MIME permitidos.

## POST /documents

Upload de documento (multipart/form-data).

| Campo | Obrigatório | Observação |
|---|---|---|
| `file` | sim | PDF, JPEG ou PNG, até `MAX_UPLOAD_MB` |
| `type` | sim | `rg`, `cpf`, `cnh`, `certidao`, `certificado`, `diploma`, `comprovante`, `outro` |
| `origin` | sim | emissor declarado (ex.: `SSP-AL`) |
| `title` | não | padrão: nome do arquivo |
| `tags` | não | separadas por vírgula ou campo repetido |
| `expires_at` | não | `YYYY-MM-DD` ou RFC3339; deve ser futuro (RF-16) |

```bash
curl -F file=@rg.pdf -F type=rg -F origin=SSP-AL -F "tags=pessoal, identidade" \
  -F expires_at=2030-12-31 \
  https://carteira.eflow.space/api/v1/documents
```

Resposta `201`:

```json
{
  "document":  { "id": "…", "owner_id": "dev-titular", "title": "rg.pdf", "type": "rg", "origin": "SSP-AL", "tags": ["pessoal","identidade"], "status": "active", "expires_at": "2030-12-31T23:59:59Z", "created_at": "…" },
  "version":   { "id": "…", "document_id": "…", "version": 1, "object_key": "documents/dev-titular/…/v1/<sha256>.pdf", "file_name": "rg.pdf", "size_bytes": 482113, "mime_type": "application/pdf", "created_at": "…" },
  "integrity": { "id": "…", "version_id": "…", "version": 1, "algorithm": "SHA-256", "hash": "<sha256>", "created_at": "…" }
}
```

O `hash` é idêntico ao de `sha256sum rg.pdf`. Documentos com `expires_at` no passado passam a `status=expired` automaticamente (job ~1 min + lazy em list/get); o hash **não** muda.

| Status | `code` | Quando |
|---|---|---|
| 400 | `invalid_form` / `missing_file` | não é multipart ou falta `file` |
| 413 | `file_too_large` | acima de `MAX_UPLOAD_MB` |
| 415 | `unsupported_type` | conteúdo não é PDF/JPEG/PNG |
| 422 | `invalid_metadata` | `type` inválido, `origin` vazio ou arquivo vazio |

## GET /documents

Lista documentos do titular (`deleted_at IS NULL`), ordenados por `created_at DESC`.

| Query | Observação |
|---|---|
| `q` | busca em título e tags (ILIKE) |
| `type` | filtro exato (`rg`, `cpf`, …) |
| `status` | filtro exato (`active`, `replaced`, …) |
| `tag` | tag exata (case-insensitive) |
| `collection` | UUID de uma coleção do titular (RF-06) |

```bash
curl 'https://carteira.eflow.space/api/v1/documents?type=rg&q=frente'
# {"items":[{…}],"total":1}
```

Cada item inclui metadados do documento + resumo da versão atual (`file_name`, `mime_type`, `size_bytes`, `version`).

## DELETE /documents/{id}

Exclusão lógica (RF-22): define `deleted_at = now()`. O documento some da listagem/detalhe; objetos MinIO e hashes **não** são apagados. Shares (RF-18) ainda não existem — a revogação de links será encaixada depois.

```bash
curl -X DELETE -i https://carteira.eflow.space/api/v1/documents/<id>
# HTTP/1.1 204 No Content
```

| Status | `code` | Quando |
|---|---|---|
| 404 | `not_found` | inexistente, já excluído ou outro titular |

## PATCH /documents/{id}

Corrige metadados sem alterar o hash/arquivo (RF-17). Body JSON:

| Campo | Observação |
|---|---|
| `title` | obrigatório (não vazio após trim; se vazio, mantém o atual) |
| `type` | um dos tipos aceitos |
| `origin` | obrigatório |
| `tags` | array de strings |
| `expires_at` | `YYYY-MM-DD` / RFC3339 futuro, ou `null` para limpar |

Permitido em `active` / `replaced` / `expired` (`expired` + validade futura/`null` → volta `active`). `revoked` → `409`.

```bash
curl -X PATCH -H 'Content-Type: application/json' \
  -d '{"title":"RG frente","type":"rg","origin":"SSP-AL","tags":["pessoal"],"expires_at":"2030-12-31"}' \
  https://carteira.eflow.space/api/v1/documents/<id>
```

Resposta `200`: documento atualizado. Auditoria `correcao_metadados` com antes → depois.

## GET /documents/{id}

Metadados + versão atual + registro de integridade. Retorna `404` se o documento não existe ou é de outro titular.

```bash
curl https://carteira.eflow.space/api/v1/documents/<id>
```

```json
{
  "document":  { "id": "…", "title": "…", "type": "rg", "status": "active", … },
  "version":   { "version": 1, "file_name": "rg.pdf", "object_key": "…", … },
  "integrity": { "algorithm": "SHA-256", "hash": "<sha256>", "version": 1, … }
}
```

## GET /documents/{id}/file

Stream do objeto no MinIO (versão atual). Por padrão `Content-Disposition: inline` (pré-visualização); use `?download=1` para forçar download.

```bash
curl -OJ 'https://carteira.eflow.space/api/v1/documents/<id>/file?download=1'
# sha256sum do arquivo baixado = integrity.hash
```

Retorna `404` se o documento não existe ou é de outro titular.

## GET /documents/{id}/receipt

PDF do recibo de integridade da **versão atual** (algoritmo, hash, ID, versão, timestamp, QR para a tela pública `/i/{id}`).

```bash
curl -OJ https://carteira.eflow.space/api/v1/documents/<id>/receipt
# Content-Type: application/pdf
```

| Status | `code` | Quando |
|---|---|---|
| 404 | `not_found` | inexistente ou outro titular |
| 422 | `missing_integrity` | documento sem registro de integridade |

## POST /documents/{id}/versions

Substitui o arquivo criando uma **nova** versão (multipart `file`). Só documentos `active`. Versões anteriores e objetos MinIO **não** são alterados.

```bash
curl -F file=@novo.pdf https://carteira.eflow.space/api/v1/documents/<id>/versions
```

Resposta `201` no mesmo formato do upload (`document` + `version` + `integrity`). Object key: `documents/{owner}/{id}/v{n}/{hash}.ext`.

| Status | `code` | Quando |
|---|---|---|
| 409 | `invalid_transition` | documento não está `active` |
| 404 | `not_found` | inexistente ou outro titular |
| 413 / 415 / 422 | iguais ao upload | tamanho / MIME / metadados |

## GET /documents/{id}/versions

Histórico de versões com hash (`version DESC`). Campo `current: true` na mais recente.

```bash
curl https://carteira.eflow.space/api/v1/documents/<id>/versions
# {"items":[{…}],"total":N}
```

## POST /documents/{id}/revoke

Revoga o documento exigindo motivo. Transição permitida: `active` ou `replaced` → `revoked`. Documento permanece listável e consultável.

```bash
curl -X POST -H 'Content-Type: application/json' \
  -d '{"reason":"emitido com nome incorreto"}' \
  https://carteira.eflow.space/api/v1/documents/<id>/revoke
```

| Status | `code` | Quando |
|---|---|---|
| 422 | `invalid_metadata` | motivo vazio |
| 409 | `invalid_transition` | já revogado/expirado |
| 404 | `not_found` | inexistente ou outro titular |

Resposta `200`: documento com `status`, `revoked_reason`, `revoked_at`.

## POST /documents/{id}/verify

Recalcula o SHA-256 do objeto no MinIO e compara com `integrity_records`. **Nunca** altera o hash registrado em caso de divergência. Grava evidência em `verifications`.

| `result` | Quando |
|---|---|
| `ok` | hashes iguais |
| `divergent` | hashes diferentes |
| `unverifiable` | falha ao ler o objeto |
| `missing_record` | sem registro de integridade |

```bash
curl -X POST https://carteira.eflow.space/api/v1/documents/<id>/verify
```

```json
{
  "id": "…",
  "document_id": "…",
  "version_id": "…",
  "version": 1,
  "result": "ok",
  "algorithm": "SHA-256",
  "expected_hash": "<sha256>",
  "computed_hash": "<sha256>",
  "context": "owner",
  "created_at": "…"
}
```

## GET /documents/{id}/verifications

Histórico de verificações (`created_at DESC`). Retorna `404` se outro titular.

```bash
curl https://carteira.eflow.space/api/v1/documents/<id>/verifications
# {"items":[{…}],"total":N}
```

## GET /documents/{id}/integrity

Registro de integridade da versão mais recente. Retorna `404` se o documento não existe ou é de outro titular.

```bash
curl https://carteira.eflow.space/api/v1/documents/<id>/integrity
```

## POST /documents/{id}/shares

Gera link de compartilhamento (RF-18/19). Body JSON `{ "days": 1|7|30 }`. Fixa a **versão atual**. Token opaco (64 hex).

```bash
curl -X POST -H 'Content-Type: application/json' -d '{"days":7}' \
  https://carteira.eflow.space/api/v1/documents/<id>/shares
```

## GET /documents/{id}/shares

Lista links do documento (titular).

## POST /shares/{id}/revoke

Revoga um link (`revoked_at=now`). Soft-delete do documento também revoga shares ativos.

## GET /public/shares/{token}

Validação por terceiro (RF-20). Não exige titular. Resposta: título, versão, `mime_type`, `file_name`, algoritmo, hash, `expires_at`, `verification` (`ok`/`divergent`/…).

```bash
curl https://carteira.eflow.space/api/v1/public/shares/<token>
```

| Status | `code` | Quando |
|---|---|---|
| 404 | `not_found` | token inexistente ou documento excluído |
| 410 | `gone` | expirado ou revogado |

URL pública da UI: `https://carteira.eflow.space/v/<token>`.

## GET /public/shares/{token}/file

Stream do arquivo da versão compartilhada (pré-visualização). `Content-Disposition: inline` por padrão; `?download=1` força download. Mesmos `404`/`410` do GET JSON.

```bash
curl -OJ 'https://carteira.eflow.space/api/v1/public/shares/<token>/file?download=1'
```

## GET /public/documents/{id}/integrity

Verificação anônima do recibo (RF-12). Não exige sessão. Recalcula o SHA-256 da versão atual e devolve título, hash, algoritmo, `registered_at` e `verification`.

```bash
curl https://carteira.eflow.space/api/v1/public/documents/<id>/integrity
```

| Status | `code` | Quando |
|---|---|---|
| 404 | `not_found` | documento inexistente ou excluído |

URL pública da UI: `https://carteira.eflow.space/i/<id>` (QR do recibo). Links antigos `/verificar?doc=<id>` redirecionam para essa tela quando não há sessão.

## GET /public/documents/{id}/file

Stream do arquivo atual para pré-visualização no recibo público. `inline` por padrão; `?download=1` força download.

```bash
curl -OJ 'https://carteira.eflow.space/api/v1/public/documents/<id>/file?download=1'
```

## GET /collections

Lista coleções do titular (RF-06), com `document_count`. Ordenadas por `created_at DESC`.

```bash
curl https://carteira.eflow.space/api/v1/collections
# {"items":[{"id":"…","name":"Pessoal","document_count":2,…}],"total":1}
```

## POST /collections

Cria coleção. Body JSON `{ "name": "…" }` (obrigatório, até 120 caracteres). Resposta `201`.

```bash
curl -X POST -H 'Content-Type: application/json' -d '{"name":"Pessoal"}' \
  https://carteira.eflow.space/api/v1/collections
```

## GET /collections/{id}

Detalhe com `document_ids` (somente documentos não excluídos do titular) e `document_count`.

## PATCH /collections/{id}

Renomeia. Body `{ "name": "…" }`.

## DELETE /collections/{id}

Remove a coleção (cascade nos vínculos). Documentos **não** são apagados. Resposta `204`.

## POST /collections/{id}/documents

Adiciona documentos à coleção (N:N). Body `{ "document_ids": ["…"] }`. Só aceita documentos do titular; IDs inválidos/outro titular → `404`. Idempotente (`ON CONFLICT DO NOTHING`).

```bash
curl -X POST -H 'Content-Type: application/json' \
  -d '{"document_ids":["<doc-uuid>"]}' \
  https://carteira.eflow.space/api/v1/collections/<id>/documents
```

## DELETE /collections/{id}/documents/{docId}

Remove o vínculo documento↔coleção. Resposta `204`.

## GET /audit

Trilha de auditoria do titular (`created_at DESC`, até 200 eventos). Ações incluem `inclusao`, `substituicao`, `revogacao`, `verificacao`, `correcao_metadados`, `exclusao`, `link_gerado`, `link_revogado`. Não inclui conteúdo de arquivo.

```bash
curl https://carteira.eflow.space/api/v1/audit
# {"items":[{"id":"…","actor":"dev-titular","action":"inclusao","document_title":"…","context":"PDF, 12 KB","created_at":"…"}],"total":N}
```

## GET /documents/{id}/assist

Status da última extração OCR + coerência assistida. Não prova autenticidade do emissor. Detalhes: [ASSISTENTE.md](./ASSISTENTE.md).

## POST /documents/{id}/assist/analyze

Dispara OCR + comparação com metadados (síncrono; pode demorar). Resposta igual ao GET. Sem `GROQ_API_KEY` → `503 assistant_unavailable`.

## POST /assistant/chat

Chat do FAB. Body `{ "messages": [{"role":"user","content":"…"}], "document_id": "…" }`. Usa RAG lexical + contexto do documento opcional.

```bash
curl -X POST -H 'Content-Type: application/json' \
  -d '{"messages":[{"role":"user","content":"O que é integridade?"}]}' \
  https://carteira.eflow.space/api/v1/assistant/chat
```


