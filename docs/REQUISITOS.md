# Levantamento de requisitos — Project LAB (Carteira)

**Tipo:** as-built (requisitos que o MVP **já cumpre** em `carteira.eflow.space`).  
**Fontes:** [API.md](./API.md), [CONTRIBUTING.md](./CONTRIBUTING.md), [CASOS_DE_USO.md](./CASOS_DE_USO.md), [ATORES.md](./ATORES.md).  
**Cards irmãos:** [REGRAS_NEGOCIO.md](./REGRAS_NEGOCIO.md); [MODULOS.md](./MODULOS.md).

## 1. Introdução

A carteira digital permite ao **titular** guardar documentos (PDF/JPEG/PNG), calcular e registrar **integridade** (SHA-256), gerir ciclo de vida e compartilhar links de verificação com **terceiros**.

**Limite explícito (RF-23):** a plataforma prova que o arquivo **não foi alterado** desde o registro; **não** prova autenticidade de órgão emissor nem assinatura digital.

**Atores:** Titular, Terceiro verificador, Sistema — ver [ATORES.md](./ATORES.md).

---

## 2. Requisitos funcionais

Status de todas as linhas: **implementado**.

### Módulo 1 — Acesso

| ID | Nome | Descrição | Critério de aceite | UC |
|---|---|---|---|---|
| RF-01 | Autenticação do titular | Cadastro e login com e-mail/senha | Register/login emite cookie; credenciais inválidas → `401` | UC01, UC02 |
| RF-02 | Gestão de sessão segura | Sessão HttpOnly; rotas privadas exigem sessão; isolamento por titular | Sem cookie → `401`; doc de outro dono → `403` | UC02, UC03 |

### Módulo 2 — Documentos

| ID | Nome | Descrição | Critério de aceite | UC |
|---|---|---|---|---|
| RF-03 | Upload/importação | Inclusão multipart PDF/JPEG/PNG com limite de tamanho | MIME inválido rejeitado; objeto no MinIO + linha em `documents` | UC04 |
| RF-04 | Metadados mínimos | Título, tipo, origem (e tags) na inclusão | Upload sem `origin`/`type` válidos → `422` | UC04 |
| RF-05 | Tipo e tags | Tipos predefinidos; tags múltiplas; filtros na listagem | Filtros `type`/`tag`/`q` na carteira | UC04, UC05 |
| RF-06 | Coleções | Agrupamento N:N documento↔coleção do titular | CRUD de coleções + filtro `?collection=` | UC13, UC05 |
| RF-07 | Listagem, busca e download | Listar/detalhar/baixar apenas documentos do titular não excluídos | `GET /documents`, `/{id}`, `/{id}/file` | UC05, UC06 |

### Módulo 3 — Integridade

| ID | Nome | Descrição | Critério de aceite | UC |
|---|---|---|---|---|
| RF-08 | Hash na inclusão | SHA-256 dos bytes exatos no upload | Hash = `sha256sum` local do arquivo enviado | UC04 |
| RF-09 | Registro de integridade | Persistência algoritmo+hash por versão | `GET …/integrity` e detalhe exibem o registro | UC04, UC07 |
| RF-10 | Verificação de integridade | Recalcular hash do objeto no MinIO e comparar | Resultado `ok`/`divergent`/…; **não** altera hash registrado | UC07 |
| RF-11 | Evidência de verificação | Histórico consultável de cada verify | `GET …/verifications` após `POST …/verify` | UC07 |
| RF-12 | Recibo de integridade | PDF com hash, metadados e QR | `GET …/receipt` → `application/pdf` | UC08 |

### Módulo 4 — Ciclo de vida

| ID | Nome | Descrição | Critério de aceite | UC |
|---|---|---|---|---|
| RF-13 | Estados do documento | `active`, `replaced`, `revoked`, `expired` visíveis e enforced | Transições inválidas → `409` | UC09, UC11 |
| RF-14 | Substituição com versionamento | Nova versão; objetos/hashes antigos imutáveis | `POST …/versions`; histórico DESC | UC09 |
| RF-15 | Revogação com motivo | Motivo obrigatório; doc permanece consultável | Sem motivo → `422`; status `revoked` | UC11 |
| RF-16 | Validade e expiração | `expires_at` opcional; expiração automática | Job/lazy → `expired` sem mudar hash | UC04, UC17 |
| RF-17 | Correção de metadados | PATCH de campos de metadados; auditável | Hash/MinIO intactos; audit `correcao_metadados` | UC10 |

### Módulo 5 — Compartilhamento

| ID | Nome | Descrição | Critério de aceite | UC |
|---|---|---|---|---|
| RF-18 | Compartilhamento controlado | Titular cria links amarrados à versão | `POST/GET …/shares` | UC14 |
| RF-19 | Token com prazo e revogação | Validade 1/7/30 dias; revoke | Expirado/revogado → `410` no público | UC14 |
| RF-20 | Validação por terceiros | Terceiro vê metadados mínimos + verify via token | Sem login; sem carteira completa | UC16 |

### Módulo 6 — Auditoria e limites de prova

| ID | Nome | Descrição | Critério de aceite | UC |
|---|---|---|---|---|
| RF-21 | Trilha de auditoria | Eventos sem conteúdo de arquivo | `GET /audit` lista ações do titular | UC15 |
| RF-22 | Exclusão lógica | Soft-delete; MinIO/hash preservados; shares revogados | `DELETE` → some da listagem; objeto permanece | UC12 |
| RF-23 | Integridade vs autenticidade | UI/API deixam claro: hash ≠ autenticidade de emissor | Painéis/AuthNote distinguem os dois conceitos | UC07, UC16 |

---

## 3. Requisitos não funcionais

| ID | Categoria | Requisito | Critério / baseline |
|---|---|---|---|
| RNF-01 | **Acessibilidade** | Formulários essenciais (login, upload, ações destrutivas) com labels; confirmação antes de revogar/excluir; erros da API legíveis na UI; rotas SPA principais usáveis por teclado | Meta acadêmica / baseline — **sem** auditoria WCAG formal neste MVP |
| RNF-02 | Segurança | Cookie de sessão HttpOnly; isolamento por `owner_id`; segredos só em `.env` (fora do git) | Cross-tenant → `403`; sem tokens em localStorage |
| RNF-03 | Disponibilidade / ops | Health agrega API + Postgres + MinIO; deploy binário único + PM2 | `GET /health` → `200`/`503` |
| RNF-04 | Desempenho / limites | Upload limitado por `MAX_UPLOAD_MB`; MIME só PDF/JPEG/PNG (magic bytes) | Arquivo grande → `413`; MIME inválido → `415` |
| RNF-05 | Manutenibilidade | Domínios `handler → service → repository`; migrations versionadas; `make ci` | Ver [CI.md](./CI.md), [ARCHITECTURE.md](./ARCHITECTURE.md) |

---

## 4. Fora do MVP

| Item | Motivo |
|---|---|
| OAuth / Login Único Gov.br | Domínio institucional `*.gov.br` + integração externa |
| Assinatura digital / autenticidade de emissor | Escopo é integridade (RF-23) |
| OCR DocTR (sidecar Python) | Pesquisa: [DOCTR.md](./DOCTR.md) — não integrado |
| Autenticidade criptográfica / órgão | Fora de escopo; coerência assistida Groq é **pós-MVP** e não substitui ([ASSISTENTE.md](./ASSISTENTE.md)) |
| Import/export interoperável com outros sistemas | Não há parceiro institucional no LAB |
| Backoffice / administrador global | Sem papéis além do titular |

---

## 5. Rastreabilidade rápida

| Artefato | Ligação |
|---|---|
| Casos de uso | [CASOS_DE_USO.md](./CASOS_DE_USO.md) |
| Atores | [ATORES.md](./ATORES.md) |
| Regras de negócio | [REGRAS_NEGOCIO.md](./REGRAS_NEGOCIO.md) |
| Módulos | [MODULOS.md](./MODULOS.md) |
| Contratos HTTP | [API.md](./API.md) |
| Mapa RF → código | [CONTRIBUTING.md](./CONTRIBUTING.md) |
