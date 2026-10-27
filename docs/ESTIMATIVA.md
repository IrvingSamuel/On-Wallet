# Estimação de prazo — Project LAB (Grupo 3)

**Método:** PERT / três pontos  
**Fórmula:** \( E = \dfrac{O + 4M + P}{6} \) (dias úteis de esforço)  
**Âncoras de complexidade (do card Chronos):**

| Complexidade | Papel no PERT | Referência |
|---|---|---|
| Baixa | O (ótimo) | ~1 dia |
| Média | M (mais provável) | ~3 dias |
| Alta | P (pessimista) | ~10 dias |

Desvio-padrão aproximado por pacote: \( \sigma = \dfrac{P - O}{6} \).

**Data da estimativa:** 26/10/2026  
**Escopo:** carteira digital com verificação de integridade (Go + React + Postgres + MinIO).  
**Equipe de referência:** 1 desenvolvedor full-time (ajuste linear se houver N pessoas em paralelo sem bloqueio).

---

## 1. Situação do quadro (Chronos)

| Status | Quantidade |
|---|---|
| Concluído | RFs 01–23 + DocTR + skill CI + estimativa + docs acadêmicas (atores, UC, requisitos, RN, módulos, UML, fluxos, glossário) |
| Pendente | *(nenhuma pendência acadêmica do quadro)* |

**RFs funcionais RF-01…RF-23:** todas `done` (código em produção em `carteira.eflow.space`).  
**Feito (processo/docs):** DocTR; skill CI; [ATORES.md](./ATORES.md); [CASOS_DE_USO.md](./CASOS_DE_USO.md); [REQUISITOS.md](./REQUISITOS.md); [REGRAS_NEGOCIO.md](./REGRAS_NEGOCIO.md); [MODULOS.md](./MODULOS.md); [UML.md](./UML.md); [FLUXOS.md](./FLUXOS.md); [GLOSSARIO.md](./GLOSSARIO.md).  
**Fora de escopo:** OAuth Gov.br (exige órgão público + domínio `*.gov.br` — inviável neste trabalho de faculdade).

---

## 2. Pacotes já entregues (retrospectiva PERT)

Estimativa *a posteriori* do esforço típico por pacote, calibrada pelo que foi implementado (migrations, handlers, UI, smoke).

| Pacote | Conteúdo | O | M | P | E | σ | Complexidade |
|---|---|---|---|---|---|---|---|
| Base + Design System + UI mock | Go boot, React, DS, SPA | 2 | 4 | 8 | **4,3** | 1,0 | Média→Alta |
| Auth / sessão (RF-01/02) | users, sessions, cookie, guard front | 1 | 2 | 5 | **2,3** | 0,7 | Média |
| Inclusão + metadados + tipos (RF-03/04/05) | upload MinIO, tags, types | 1 | 3 | 6 | **3,2** | 0,8 | Média |
| Carteira / listagem (RF-07) | GET list/detail/file + Wallet | 1 | 2 | 4 | **2,2** | 0,5 | Baixa→Média |
| Integridade (RF-08/09/10/11) | hash, verify, evidências | 2 | 4 | 8 | **4,3** | 1,0 | Média→Alta |
| Recibo PDF (RF-12) | gofpdf + QR | 0,5 | 1,5 | 3 | **1,6** | 0,4 | Baixa→Média |
| Estados / revogação / validade (RF-13/15/16) | status, revoke, expires job | 1 | 3 | 6 | **3,2** | 0,8 | Média |
| Versionamento (RF-14) | POST/GET versions | 1 | 2 | 4 | **2,2** | 0,5 | Média |
| Metadados + exclusão (RF-17/22) | PATCH, soft-delete | 1 | 2 | 4 | **2,2** | 0,5 | Média |
| Shares + link público (RF-18/19/20) | tokens, /v/:token | 2 | 3 | 7 | **3,5** | 0,8 | Média→Alta |
| Auditoria (RF-21) | audit_events + UI | 0,5 | 1,5 | 3 | **1,6** | 0,4 | Baixa→Média |
| Coleções (RF-06) | N:N + filtro Wallet | 1 | 2 | 4 | **2,2** | 0,5 | Média |
| UI integridade×autenticidade (RF-23) | AuthNote / painéis | 0,5 | 1 | 2 | **1,1** | 0,3 | Baixa |
| **Subtotal entregue** | | | | | **≈ 34,0** | | |

Interpretação: o núcleo de software do MVP (~34 dias-pessoa PERT) já foi consumido. Em calendário com 1 pessoa e interrupções acadêmicas, isso corresponde a várias semanas de sprint contínuo — coerente com o histórico do quadro (ago→out/2026).

---

## 3. Trabalho restante (estimativa prospectiva)

### 3.1 Documentação acadêmica

| Pacote | Itens Chronos | O | M | P | E | σ |
|---|---|---|---|---|---|---|
| Requisitos + RN + módulos | Levantamento de requisitos; Definição de Regra de Negócio; Definição dos Módulos | 1 | 2 | 4 | **2,2** | 0,5 |
| Atores + matriz + casos de uso | Identificar atores; Matriz de Atores; Diagrama de caso de uso | 1 | 2 | 5 | **2,3** | 0,7 |
| UML + fluxos + glossário | Modelagem UML; Fluxos detalhados; Glossário de Entidades | 1,5 | 3 | 6 | **3,3** | 0,8 |
| **Subtotal docs** | | | | | **≈ 7,8** | |

### 3.2 Pesquisa / processo

| Pacote | Item Chronos | O | M | P | E | σ | Nota |
|---|---|---|---|---|---|---|---|
| Avaliação DocTR | Verificar uso do DocTR | 0,5 | 1 | 3 | **1,3** | 0,4 | Decisão go/no-go OCR; **não** inclui integração completa |
| Skill CI / branches | Criar skill para sempre usar CI completo | 0,5 | 1,5 | 4 | **1,8** | 0,6 | Skill + convenção main/dev/release; CI real depende do remoto Git |
| **Subtotal processo** | | | | | **≈ 3,0** | |

### 3.3 Opcional (fora do quadro atual, se DocTR for adotado)

| Pacote | O | M | P | E | Observação |
|---|---|---|---|---|---|
| Integração OCR DocTR no upload | 3 | 6 | 12 | **6,5** | Serviço Python/sidecar, fila, UI de revisão — alta incerteza |

---

## 4. Totais e prazos recomendados

| Cenário | Esforço PERT (dias-pessoa) | Calendário (1 pessoa, ~4–5 dias úteis/semana) |
|---|---|---|
| **Só fechar pendências do quadro** (docs + DocTR pesquisa + skill CI) | \(7{,}8 + 3{,}0 \approx\) **10,8** | **≈ 2,5 a 3 semanas** |
| Buffer 20% (reuniões, revisão do professor) | **≈ 13** | **≈ 3 semanas** |
| Se incluir integração DocTR | \(10{,}8 + 6{,}5 \approx\) **17,3** (+20% → ~21) | **≈ 4 a 5 semanas** |

**MVP de software:** já entregue (RF-01…23 + coleções + auth).  
**Entrega acadêmica completa do quadro:** foco nos **~11 dias-pessoa** restantes de documentação/processo.

### Prazo sugerido para o grupo

| Marco | Meta |
|---|---|
| Documentação (requisitos, atores, UML, RN, glossário, fluxos, módulos) | **+10 dias úteis** a partir de 27/10/2026 → **≈ 10/11/2026** |
| DocTR (parecer escrito) + skill CI | em paralelo na mesma janela (**+3 dias** espalhados) |
| Buffer / revisão final | até **≈ 14/11/2026** |

---

## 5. Premissas e riscos

1. Estimativas em **dias-pessoa**; paralelizar entre 4 membros reduz calendário, não o esforço total (overhead de sync ~10–20%).
2. DocTR (https://github.com/mindee/doctr) é **pesquisa de viabilidade**; integração completa não está no escopo mínimo do card.
3. Branches protegidas / CI completo exigem repositório Git remoto configurado — o card de skill documenta o fluxo; a ativação no GitHub/GitLab é pré-requisito.
4. Risco alto residual: mudança de escopo do professor (ex.: exigir OCR na demo) → usar linha “opcional DocTR” da §3.3.
5. Software já em produção: regressões e polish de UX **não** estão nestes totais (reservar 1–2 dias se a demo exigir).

---

## 6. Resumo executivo

- **Feito (software):** ~**34** dias-pessoa PERT (RF-01…23 + base).  
- **Falta (quadro):** ~**11** dias-pessoa PERT → **~3 semanas** com 1 pessoa ou **~1–1,5 semana** com 3–4 pessoas em docs paralelas.  
- **Fórmula aplicada:** \(E=(O+4M+P)/6\) com âncoras 1 / 3 / 10 dias por complexidade.
