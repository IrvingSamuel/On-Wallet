# Fluxos detalhados — Project LAB (Carteira)

**Tipo:** as-built (sequências Mermaid dos fluxos críticos).  
**Fontes:** [ARCHITECTURE.md](./ARCHITECTURE.md), [API.md](./API.md), [CASOS_DE_USO.md](./CASOS_DE_USO.md), [REGRAS_NEGOCIO.md](./REGRAS_NEGOCIO.md), [UML.md](./UML.md).

## Índice

| ID | Fluxo | UC | RF / RN |
|---|---|---|---|
| F1 | Autenticar / sessão | UC01–03 | RF-01/02 · RN-01…04 |
| F2 | Incluir documento (upload + integridade) | UC04 | RF-03/08/09 · RN-05…09 |
| F3 | Verificar integridade | UC07 | RF-10/11/23 · RN-24/25 |
| F4 | Substituir versão | UC09 | RF-14 · RN-10 |
| F5 | Revogar documento | UC11 | RF-13/15 · RN-12/13 |
| F6 | Criar share + validar link | UC14, UC16 | RF-18–20 · RN-19…22 |
| F7 | Expiração automática | UC17 | RF-16 · RN-14 |
| F8 | Exclusão lógica (+ revoga shares) | UC12 | RF-22 · RN-17/18 |

---

## F1 — Autenticar / sessão

```mermaid
sequenceDiagram
  actor Titular
  participant SPA
  participant API
  participant PG as Postgres

  Titular->>SPA: email senha
  SPA->>API: POST /auth/login
  API->>PG: FindByEmail + bcrypt
  alt credenciais OK
    API->>PG: INSERT sessions token_hash
    API-->>SPA: 200 Set-Cookie carteira_session
    SPA-->>Titular: carteira
  else invalido
    API-->>SPA: 401
  end
```

**Pré:** conta existente (ou register UC01). **Pós:** cookie HttpOnly. **Exceções:** `401` credenciais; `422` input.

---

## F2 — Incluir documento (upload + integridade)

```mermaid
sequenceDiagram
  actor Titular
  participant SPA
  participant API
  participant MinIO
  participant PG as Postgres

  Titular->>SPA: arquivo e metadados
  SPA->>API: POST /documents multipart cookie
  API->>API: MIME tamanho type origin
  API->>API: SHA256
  API->>MinIO: Put object_key
  API->>PG: BEGIN documents versions integrity
  alt commit OK
    PG-->>API: commit
    API-->>SPA: 201
  else falha DB
    API->>MinIO: Remove orfao
    API-->>SPA: 5xx
  end
```

**Exceções:** `401`; `413`; `415`; `422`.

---

## F3 — Verificar integridade

```mermaid
sequenceDiagram
  actor Titular
  participant SPA
  participant API
  participant MinIO
  participant PG as Postgres

  Titular->>SPA: Verificar
  SPA->>API: POST /documents/id/verify cookie
  API->>PG: versao + integrity_records
  API->>MinIO: Get object_key
  API->>API: SHA256 comparado
  API->>PG: INSERT verifications audit
  API-->>SPA: ok ou divergent
  SPA-->>Titular: painel + AuthNote RF23
```

**Nota:** hash registrado **não** é sobrescrito (RN-24).

---

## F4 — Substituir versão

```mermaid
sequenceDiagram
  actor Titular
  participant SPA
  participant API
  participant MinIO
  participant PG as Postgres

  Titular->>SPA: Substituir arquivo
  SPA->>API: POST /documents/id/versions multipart
  API->>API: valida MIME hash
  API->>MinIO: Put nova object_key
  API->>PG: INSERT version N + integrity
  API->>PG: status replaced na versao anterior se aplicavel
  API-->>SPA: 201 nova versao
```

**Pré:** documento do titular elegível. **Pós:** versão N imutável; hash próprio (RN-10).

---

## F5 — Revogar documento

```mermaid
sequenceDiagram
  actor Titular
  participant SPA
  participant API
  participant PG as Postgres

  Titular->>SPA: motivo da revogacao
  SPA->>API: POST /documents/id/revoke JSON
  API->>API: exige motivo
  API->>PG: status revoked revoked_at reason
  API->>PG: audit
  API-->>SPA: 200 documento
```

**Pré:** `active` ou `replaced`. **Exceções:** motivo vazio → `422`; transição inválida → `409`. Hash intacto (RN-13).

---

## F6 — Criar share + validar link

```mermaid
sequenceDiagram
  actor Titular
  actor Terceiro
  participant SPA
  participant API
  participant PG as Postgres

  Titular->>SPA: criar link 1_7_30 dias
  SPA->>API: POST /documents/id/shares cookie
  API->>PG: INSERT shares token
  API-->>SPA: URL /v/token

  Terceiro->>SPA: abre /v/token
  SPA->>API: GET /public/shares/token
  alt valido
    API-->>SPA: metadados publicos
  else expirado ou revogado
    API-->>SPA: 410
  end
```

---

## F7 — Expiração automática (Sistema)

```mermaid
flowchart TD
  startNode[Job_1min_ou_lazy]
  check{expires_at_passado_e_nao_expired}
  setExpired[status_expired]
  keepHash[hash_e_MinIO_intactos]
  endNode[Fim]

  startNode --> check
  check -->|sim| setExpired --> keepHash --> endNode
  check -->|nao| endNode
```

Ator: **Sistema** (UC17). Não altera prova de integridade (RN-13/14).

---

## F8 — Exclusão lógica

```mermaid
sequenceDiagram
  actor Titular
  participant SPA
  participant API
  participant PG as Postgres

  Titular->>SPA: Excluir confirma
  SPA->>API: DELETE /documents/id cookie
  API->>PG: SET deleted_at
  API->>PG: revoga shares ativos
  API->>PG: audit exclusao
  API-->>SPA: 204
```

**Nota:** sem `Remove` no MinIO (RN-17/18).

---

## Rastreabilidade FE

| Fluxo | Páginas |
|---|---|
| F1 | `Login.tsx`, `auth.tsx` |
| F2 | `Upload.tsx` |
| F3 | `DocumentDetail.tsx`, `VerifyIntegrity.tsx` |
| F4–F5, F8 | `DocumentDetail.tsx` |
| F6 | `Share.tsx`, `Verify.tsx` |
| F7 | job backend + lazy list/get |
