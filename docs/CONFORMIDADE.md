# Parecer de conformidade — Project LAB (Carteira)

**Data:** 26/10/2026  
**Ambiente:** https://carteira.eflow.space  
**Fontes:** [REQUISITOS.md](./REQUISITOS.md), [REGRAS_NEGOCIO.md](./REGRAS_NEGOCIO.md), cards Chronos RF-01…23, smoke API ao vivo.

> A ementa/PDF original do professor **não** está versionada no repositório. Este parecer usa o levantamento as-built alinhado ao material *Carteira Digital* / quadro Chronos (Grupo 3, UNIMA Afya).

## Veredito

**O MVP atende RF-01…23 e RN-01…10.** Autenticidade de órgão emissor permanece **fora de escopo** (RF-23 / RN-05), de propósito. Capacidades pós-MVP (assistente OCR/IA, visualizador PDF próprio) **não** alteram esse limite.

| Bloco | Resultado |
|---|---|
| RF funcionais | **23/23** OK |
| Regras de negócio | **10/10** OK |
| RNF | **5/5** (RNF-01 = baseline acadêmico, sem WCAG formal) |
| Health produção | `postgres` + `minio` **ok** |

## Smoke ao vivo (26/10/2026 ~23:20 −03)

| Critério | Esperado | Obtido |
|---|---|---|
| `GET /health` | ok | ok |
| Register + `/auth/me` | 201 / 200 + cookie | OK |
| `GET /documents` sem cookie | 401 | 401 |
| Upload PNG; hash API vs `sha256sum` | iguais | **YES** |
| `POST …/verify` | `ok` | `ok` |
| Doc de outro titular | 403 | 403 |
| Upload `.txt` | 415 | 415 |
| Revogar com `reason=""` | 422 | 422 |
| Share público | verification ok | ok |
| `GET …/receipt` | `application/pdf` | `%PDF-` |
| Soft-delete | 204 → GET 404 | OK |
| Share após delete | inválido | 404 |

## Matriz RF (resumo)

Todos **implementados** com API + UI reais — ver mapa em [CONTRIBUTING.md](./CONTRIBUTING.md).

Observação RF-13/14: após substituição, o documento permanece `status=active` e o histórico fica em `document_versions` (imutabilidade = RN-06). O valor `replaced` existe no enum e nas transições de revoke/expire.

## Matriz RN

| RN | Status |
|---|---|
| RN-01 Isolamento | OK (401/403) |
| RN-02 Formato/tamanho | OK (415/413) |
| RN-03 Hash na TX | OK |
| RN-04 Divergência não sobrescreve | OK (código Verify) |
| RN-05 Integridade ≠ autenticidade | OK (UI + textos públicos) |
| RN-06 Versões imutáveis | OK |
| RN-07 Revogação com motivo | OK |
| RN-08 Expiração sem mudar hash | OK (job + lazy) |
| RN-09 Share prazo/revogação | OK |
| RN-10 Soft-delete | OK |

## Fora do MVP (conforme ementa / escopo)

- OAuth Gov.br  
- Autenticidade criptográfica / órgão emissor  
- DocTR (sidecar Python)  
- Admin global  
- Interoperabilidade institucional  

**Pós-MVP (extra):** assistente OCR/Tesseract + Groq (coerência assistida), FAB, visualizador PDF.js — ver [ASSISTENTE.md](./ASSISTENTE.md).

## Pendências menores (não bloqueiam)

1. Checklist Chronos do card de casos de uso: “revisar fluxo com stakeholders” ainda aberto (processo).  
2. RNF-01 sem laudo WCAG formal (já declarado no levantamento).

## Artefatos de rastreabilidade

ATORES · CASOS_DE_USO · REQUISITOS · REGRAS_NEGOCIO · MODULOS · UML · FLUXOS · GLOSSARIO · API · ARCHITECTURE · CONTRIBUTING · ASSISTENTE · este CONFORMIDADE.
