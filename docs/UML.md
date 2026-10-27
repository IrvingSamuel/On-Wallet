# Modelagem UML — Project LAB (Carteira)

**Tipo:** as-built (estrutura do MVP em Mermaid).  
**Fontes:** [GLOSSARIO.md](./GLOSSARIO.md), [MODULOS.md](./MODULOS.md), [ATORES.md](./ATORES.md), [CASOS_DE_USO.md](./CASOS_DE_USO.md).  
**Casos de uso:** diagrama e catálogo UC01–UC17 em [CASOS_DE_USO.md](./CASOS_DE_USO.md) (não duplicados aqui).  
**Demais sequências:** [FLUXOS.md](./FLUXOS.md).

## 1. Introdução

Modelagem estrutural do domínio persistido e da implantação lógica (SPA + API Go + Postgres + MinIO). Classes espelham as entidades do glossário; componentes espelham os módulos M1–M6. Inclui diagrama de estados do documento e uma sequência de referência (upload).

---

## 2. Diagrama de classes (domínio)

```mermaid
classDiagram
  direction TB

  class User {
    +UUID id
    +string email
    +string passwordHash
    +string name
    +DateTime createdAt
  }

  class Session {
    +UUID id
    +UUID userId
    +string tokenHash
    +DateTime expiresAt
  }

  class Document {
    +UUID id
    +string ownerId
    +string title
    +string type
    +string origin
    +string[] tags
    +Status status
    +DateTime expiresAt
    +string revokedReason
    +DateTime revokedAt
    +DateTime deletedAt
  }

  class DocumentVersion {
    +UUID id
    +UUID documentId
    +int version
    +string objectKey
    +string fileName
    +long sizeBytes
    +string mimeType
  }

  class IntegrityRecord {
    +UUID id
    +UUID versionId
    +string algorithm
    +string hash
  }

  class Verification {
    +UUID id
    +UUID documentId
    +UUID versionId
    +VerifyResult result
    +string expectedHash
    +string computedHash
    +string context
  }

  class Share {
    +UUID id
    +string ownerId
    +UUID documentId
    +UUID versionId
    +string token
    +DateTime expiresAt
    +DateTime revokedAt
  }

  class Collection {
    +UUID id
    +string ownerId
    +string name
  }

  class CollectionDocument {
    +UUID collectionId
    +UUID documentId
    +DateTime addedAt
  }

  class AuditEvent {
    +UUID id
    +string ownerId
    +string actor
    +string action
    +UUID documentId
    +string documentTitle
    +string context
  }

  class Status {
    <<enumeration>>
    active
    replaced
    revoked
    expired
  }

  class VerifyResult {
    <<enumeration>>
    ok
    divergent
    unverifiable
    missing_record
  }

  User "1" --> "*" Session : sessions
  User "1" --> "*" Document : owns
  Document "1" --> "*" DocumentVersion : versions
  DocumentVersion "1" --> "1" IntegrityRecord : proves
  Document "1" --> "*" Verification : checks
  DocumentVersion "1" --> "*" Verification : ofVersion
  Document "1" --> "*" Share : shares
  DocumentVersion "1" --> "*" Share : pinnedVersion
  Document "1" --> "*" AuditEvent : logged
  Collection "1" --> "*" CollectionDocument
  Document "1" --> "*" CollectionDocument
  Document --> Status
  Verification --> VerifyResult
```

Notas:

- `ownerId` nas entidades de carteira = `User.id` (texto no schema).
- `CollectionDocument` é a associação N:N explícita (tabela `collection_documents`).
- Bytes do arquivo **não** são atributos de classe de domínio; vivem no MinIO via `objectKey` (RN-03/06).

---

## 3. Diagrama de componentes

```mermaid
flowchart TB
  subgraph client [Cliente]
    spa[SPA_React]
  end

  subgraph server [Binario_Go]
    api[HTTP_API_v1]
    authPkg[internal_auth_M1]
    docsPkg[internal_documents_M2_M3_M4_M6]
    collPkg[internal_collections_M2]
    sharePkg[internal_shares_M5]
    webEmbed[embed_FS_SPA]
  end

  subgraph data [Persistencia]
    pg[(Postgres)]
    minio[(MinIO)]
  end

  spa -->|"cookie_carteira_session"| api
  spa -.->|"assets"| webEmbed
  api --> authPkg
  api --> docsPkg
  api --> collPkg
  api --> sharePkg
  authPkg --> pg
  docsPkg --> pg
  docsPkg --> minio
  collPkg --> pg
  sharePkg --> pg
```

| Componente | Módulo lógico | Pacote / superfície |
|---|---|---|
| `internal/auth` | M1 Acesso | register/login/logout/me + middleware |
| `internal/documents` | M2+M3+M4+M6 | upload, verify, versions, ciclo de vida, `GET /audit` |
| `internal/collections` | M2 (coleções) | CRUD + membership |
| `internal/shares` | M5 | create/list/revoke + público |
| SPA React | UI | páginas por UC ([CASOS_DE_USO.md](./CASOS_DE_USO.md) §4) |

M6 (auditoria) é módulo lógico; no código as-built os eventos e `GET /audit` ficam em `internal/documents` (ou pacote `internal/audit` conforme versão do código).

---

## 4. Diagrama de estados — Document

```mermaid
stateDiagram-v2
  [*] --> active: upload_v1

  active --> replaced: nova_versao
  active --> revoked: revogar_com_motivo
  active --> expired: expires_at_passado

  replaced --> revoked: revogar_com_motivo
  replaced --> expired: expires_at_passado

  revoked --> [*]
  expired --> [*]

  note right of active
    Soft-delete: deleted_at
    independente do status
    (some da listagem)
  end note
```

Regras: `active`/`replaced` → `revoked` (RN-12); expiração não altera hash (RN-13); soft-delete revoga shares (RN-18).

---

## 5. Sequência de referência — upload + hash

```mermaid
sequenceDiagram
  actor Titular
  participant API
  participant MinIO
  participant PG as Postgres

  Titular->>API: POST /documents multipart cookie
  API->>API: MIME tamanho type origin
  API->>API: SHA256
  API->>MinIO: Put object_key
  API->>PG: TX documents versions integrity
  alt OK
    PG-->>API: commit
    API-->>Titular: 201
  else falha DB
    API->>MinIO: Remove orfao
    API-->>Titular: 5xx
  end
```

Detalhe e demais fluxos: [FLUXOS.md](./FLUXOS.md) (F0–F7).

---

## 6. Rastreabilidade

| Artefato UML | Doc |
|---|---|
| Classes | [GLOSSARIO.md](./GLOSSARIO.md) |
| Estados | §4 + RF-13/15/16 |
| Sequência upload | §5 + [FLUXOS.md](./FLUXOS.md) F1 |
| Componentes / módulos | [MODULOS.md](./MODULOS.md) |
| Casos de uso | [CASOS_DE_USO.md](./CASOS_DE_USO.md) |
| Atores | [ATORES.md](./ATORES.md) |
| Implantação física | [ARCHITECTURE.md](./ARCHITECTURE.md) |
