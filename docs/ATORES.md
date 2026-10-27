# Atores do sistema

**Projeto:** Carteira digital de documentos (`carteira.eflow.space`)  
**Escopo:** MVP RF-01…RF-23 em produção (Go + React + Postgres + MinIO).  
**Casos de uso:** ver [CASOS_DE_USO.md](./CASOS_DE_USO.md).

## 1. Atores canônicos

| Ator | Tipo | Identificação | Resumo |
|---|---|---|---|
| **Titular** | Humano | Sessão cookie HttpOnly `carteira_session` após register/login (`internal/auth`) | Dono dos documentos; opera a carteira autenticada. |
| **Terceiro verificador** | Humano | Token opaco de compartilhamento (`GET /public/shares/{token}` ou rota SPA `/v/:token`) | Consulta somente o que o link público permite; sem conta. |
| **Sistema** | Não humano | Processos do servidor (job de expiração ~1 min, lazy em list/get, middleware) | Aplica regras automáticas (expiração, isolamento, soft-delete). |

### 1.1 Titular

- Cadastra-se e autentica com e-mail e senha (RF-01/02). Não há Login Único Gov.br neste trabalho acadêmico (domínio `*.gov.br` exigido pelo provedor oficial).
- Inclui, lista, busca, baixa, verifica integridade, substitui versões, corrige metadados, revoga, exclui logicamente, organiza em coleções, gera/revoga links e consulta a própria trilha de auditoria.
- Isolamento: toda consulta privada filtra por `owner_id` (= UUID do usuário). Documento de outro titular → `403`.

### 1.2 Terceiro verificador

- Acessa apenas via link/token válido, não expirado e não revogado (RF-18/19/20).
- Pode inspecionar metadados públicos do share e o resultado de verificação associado ao token — **não** autentica, **não** altera documentos, **não** vê a carteira completa do titular.
- Se o documento foi excluído logicamente ou o share foi revogado/expirou, o acesso público falha.

### 1.3 Sistema

- Expira documentos com `expires_at` no passado → `status=expired` (RF-16), sem alterar o hash.
- Na exclusão lógica, revoga shares ativos do documento (RF-22).
- Middleware de sessão rejeita rotas privadas sem cookie válido (`401`).
- Health check agrega Postgres + MinIO.

## 2. O que não existe neste MVP

| Papel hipotético | Motivo |
|---|---|
| Administrador global / backoffice | Não há papéis além do titular da própria conta. |
| Órgão emissor / autoridade certificadora | A carteira prova **integridade** do arquivo (hash); **não** prova autenticidade do emissor (RF-23). |
| Cidadão via OAuth Gov.br | Fora de escopo acadêmico (pré-requisito institucional + domínio `*.gov.br`). |
| Usuário anônimo na API privada | Sem sessão → `401` (exceto health, auth register/login e shares públicos). |

## 3. Matriz de atores × capacidades

Legenda: **S** = sim · **—** = não · **A** = automático (Sistema)

| Capacidade | RF | Titular | Terceiro | Sistema |
|---|---|---|---|---|
| Cadastrar / autenticar / encerrar sessão | 01/02 | S | — | — |
| Upload + metadados + hash SHA-256 | 03/04/08/09 | S | — | — |
| Classificar tipo e tags | 05 | S | — | — |
| Gerir coleções | 06 | S | — | — |
| Listar, buscar, detalhe, download | 07 | S | —¹ | — |
| Verificar integridade + evidência | 10/11 | S | —¹ | — |
| Recibo PDF de integridade | 12 | S | — | — |
| Estados do documento | 13 | S | — | A (expired) |
| Nova versão (substituição) | 14 | S | — | — |
| Revogar com motivo | 15 | S | — | — |
| Definir validade / expirar | 16 | S | — | A |
| Corrigir metadados (auditável) | 17 | S | — | — |
| Criar / listar / revogar shares | 18/19 | S | — | A² |
| Validar documento via link | 20 | — | S | — |
| Consultar auditoria própria | 21 | S | — | A (grava eventos) |
| Exclusão lógica | 22 | S | — | A² |
| Distinção integridade × autenticidade (UI) | 23 | S | S³ | — |

¹ Terceiro só vê o subconjunto exposto pelo endpoint público do token, não `GET /documents`.  
² Na exclusão lógica o Sistema revoga shares ativos.  
³ A tela pública `/v/:token` também deixa claro que integridade ≠ autenticidade do emissor.

## 4. Relação com pacotes de código

| Ator | Pacotes / superfícies |
|---|---|
| Titular | `internal/auth`, `internal/documents`, `internal/collections`, `internal/shares`, `internal/audit`; SPA autenticada |
| Terceiro | `GET /api/v1/public/shares/{token}`; `frontend` rota `/v/:token` |
| Sistema | job `ExpireDue` em documents; middleware de sessão; hooks de SoftDelete → ShareRevoker |

Detalhe de rotas: [API.md](./API.md). Mapa RF → código: [CONTRIBUTING.md](./CONTRIBUTING.md).
