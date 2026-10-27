# Guia para a equipe

Estimativa de prazo (PERT): ver [ESTIMATIVA.md](./ESTIMATIVA.md).  
Parecer OCR/DocTR: ver [DOCTR.md](./DOCTR.md). Assistente OCR+IA+RAG: [ASSISTENTE.md](./ASSISTENTE.md).  
Conformidade MVP (RF/RN): [CONFORMIDADE.md](./CONFORMIDADE.md).  
CI / branches / PRs / releases: ver [CI.md](./CI.md) (comando: `make ci`; skill Cursor: `.cursor/skills/ci-completo/`).  
Atores e casos de uso: [ATORES.md](./ATORES.md), [CASOS_DE_USO.md](./CASOS_DE_USO.md).  
Levantamento de requisitos (RF + RNF/acessibilidade): [REQUISITOS.md](./REQUISITOS.md).  
Regras de negócio (RN-01…10): [REGRAS_NEGOCIO.md](./REGRAS_NEGOCIO.md).  
Módulos do sistema: [MODULOS.md](./MODULOS.md).  
Glossário de entidades: [GLOSSARIO.md](./GLOSSARIO.md).  
Modelagem UML (classes + componentes): [UML.md](./UML.md).  
Fluxos detalhados (sequências): [FLUXOS.md](./FLUXOS.md).

## Estado do MVP

Todas as partes abaixo estão **reais** (API + UI; sem mock/`localStorage`).

| Parte | Estado |
|---|---|
| Upload para o MinIO + metadados mínimos (RF-03/04) | real |
| SHA-256 + registro de integridade (RF-08/09) | real |
| Listagem, busca, detalhe e download (RF-07) | real (`GET /documents`, `/{id}`, `/{id}/file`) |
| Verificação de integridade + evidência (RF-10/11) | real (`POST …/verify`, `GET …/verifications`) |
| Estados + revogação com motivo (RF-13/15) | real (`POST …/revoke`) |
| Tipos e tags (RF-05) | real (upload + filtros de listagem) |
| Telas Carteira, Detalhe, Novo documento, Recibo e Verificar | real |
| Substituição com versionamento (RF-14) | real (`POST …/versions`, `GET …/versions`) |
| Recibo de integridade em PDF (RF-12) | real (`GET …/receipt`) |
| Validade e expiração automática (RF-16) | real (`expires_at` + job/lazy → `expired`) |
| Correção de metadados auditável (RF-17) | real (`PATCH …/documents/{id}`) |
| Exclusão lógica (RF-22) | real (`DELETE …/documents/{id}` → `deleted_at`) |
| Compartilhamento + validação por link (RF-18/19/20) | real (`…/shares`, `GET /public/shares/{token}`) |
| Autenticação e sessão (RF-01/02) | real (`/auth/*`, cookie `carteira_session`, middleware) |
| Coleções (RF-06) | real (`/collections`, filtro `?collection=` na carteira) |
| Auditoria (RF-21) | real (`GET /audit`, tabela `audit_events`) |

## Onde cada RF entra

| RF | Backend | Frontend |
|---|---|---|
| RF-01/02 autenticação e sessão | **feito** — `internal/auth` (users/sessions); cookie + middleware; cross-tenant → 403 | `Login.tsx`, `auth.tsx` guard |
| RF-05 tipos e tags | **feito** — `documents.Types` + tags no upload; filtros `type`/`tag`/`q` | `Upload.tsx`, `Wallet.tsx` |
| RF-06 coleções | **feito** — `collections` + `collection_documents`; CRUD + membership | `Collections.tsx`, filtro em `Wallet.tsx` |
| RF-07 listagem, busca, download | **feito** — `GET /documents`, `GET /documents/{id}`, `GET /documents/{id}/file` | `Wallet.tsx`, `DocumentDetail.tsx`, `Receipt.tsx` |
| RF-10/11 verificação + evidência | **feito** — `POST /documents/{id}/verify`, `GET …/verifications`, tabela `verifications` | `DocumentDetail.tsx`, `VerifyIntegrity.tsx` |
| RF-12 recibo | **feito** — `GET /documents/{id}/receipt` (PDF + QR) | `Receipt.tsx` Baixar PDF |
| RF-13 estados | **feito** (base) — transições via `Service` (`active`/`replaced` → `revoked`) | `StatusBadge` |
| RF-14 versionamento | **feito** — `POST /documents/{id}/versions`, `GET …/versions` (versões antigas imutáveis) | `DocumentDetail.tsx` Substituir |
| RF-15 revogação com motivo | **feito** — `POST /documents/{id}/revoke`, colunas `revoked_reason`/`revoked_at` | `DocumentDetail.tsx` |
| RF-16 expiração | **feito** — `expires_at` + job ~1 min + lazy em list/get | `Upload.tsx` validade; detalhe |
| RF-17 correção de metadados | **feito** — `PATCH /documents/{id}` + audit `correcao_metadados`; nunca mexe no hash | `DocumentDetail.tsx` Editar |
| RF-18/19/20 compartilhamento | **feito** — tabela `shares`; create/list/revoke; `GET /public/shares/{token}` | `Share.tsx`, `Verify.tsx` (`/v/:token`) |
| RF-21 auditoria | **feito** — `GET /audit`, tabela `audit_events` (incl. `link_gerado`/`link_revogado`) | `pages/Audit.tsx` |
| RF-22 exclusão lógica | **feito** — `DELETE /documents/{id}` + audit `exclusao`; revoga shares ativos | `DocumentDetail.tsx` Excluir |
| RF-23 integridade x autenticidade | **feito** (UI) — painel + `AuthNote` + coerência assistida (não prova emissor) | `DocumentDetail.tsx`, `VerifyIntegrity.tsx`, `AssistantFab.tsx` |
| Pós-MVP assistente OCR/IA/RAG | **feito** — Groq + FTS; ver [ASSISTENTE.md](./ASSISTENTE.md) | FAB + `POST /assistant/chat` |

## Convenções

- Um pacote por domínio em `backend/internal/`, seguindo `handler → service → repository`, como em `internal/documents`.
- Rotas registradas com os padrões do `http.ServeMux` (`"POST /api/v1/..."`), sem framework.
- Mudança de schema = novo arquivo `backend/internal/db/migrations/000N_descricao.sql`. Nunca edite uma migration já aplicada.
- Toda consulta filtra por `owner_id` (isolamento entre titulares, RF-02).
- Segredos só no `.env`, que não vai para o git.

## Deploy

```bash
make build && pm2 restart carteira
curl https://carteira.eflow.space/api/v1/health
```
