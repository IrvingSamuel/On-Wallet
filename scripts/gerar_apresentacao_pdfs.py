#!/usr/bin/env python3
"""Gera PDFs de apresentacao individual e geral — Project LAB Grupo 3 (On-Wallet)."""

from __future__ import annotations

from pathlib import Path

from reportlab.lib import colors
from reportlab.lib.enums import TA_CENTER, TA_JUSTIFY, TA_LEFT
from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import ParagraphStyle, getSampleStyleSheet
from reportlab.lib.units import cm
from reportlab.platypus import (
    ListFlowable,
    ListItem,
    PageBreak,
    Paragraph,
    SimpleDocTemplate,
    Spacer,
    Table,
    TableStyle,
)

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "docs" / "apresentacao"
PRIMARY = colors.HexColor("#1F4F8A")
ACCENT = colors.HexColor("#2966B1")
MUTED = colors.HexColor("#5B6676")
LIGHT = colors.HexColor("#EAF1FA")


def styles():
    base = getSampleStyleSheet()
    return {
        "cover_title": ParagraphStyle(
            "cover_title", parent=base["Title"], fontSize=22, textColor=PRIMARY,
            alignment=TA_CENTER, spaceAfter=12, leading=28,
        ),
        "cover_sub": ParagraphStyle(
            "cover_sub", parent=base["Normal"], fontSize=12, textColor=MUTED,
            alignment=TA_CENTER, spaceAfter=6, leading=16,
        ),
        "h1": ParagraphStyle(
            "h1", parent=base["Heading1"], fontSize=14, textColor=PRIMARY,
            spaceBefore=14, spaceAfter=8, leading=18,
        ),
        "h2": ParagraphStyle(
            "h2", parent=base["Heading2"], fontSize=12, textColor=ACCENT,
            spaceBefore=10, spaceAfter=6, leading=15,
        ),
        "body": ParagraphStyle(
            "body", parent=base["Normal"], fontSize=10, leading=14,
            alignment=TA_JUSTIFY, spaceAfter=6,
        ),
        "bullet": ParagraphStyle(
            "bullet", parent=base["Normal"], fontSize=10, leading=13,
            leftIndent=4, spaceAfter=2,
        ),
        "meta": ParagraphStyle(
            "meta", parent=base["Normal"], fontSize=9, textColor=MUTED,
            alignment=TA_CENTER, spaceAfter=4,
        ),
        "path": ParagraphStyle(
            "path", parent=base["Code"], fontSize=8, textColor=PRIMARY,
            leading=11, spaceAfter=2,
        ),
        "footer": ParagraphStyle(
            "footer", parent=base["Normal"], fontSize=8, textColor=MUTED,
            alignment=TA_CENTER,
        ),
    }


def bullets(items: list[str], st) -> ListFlowable:
    return ListFlowable(
        [ListItem(Paragraph(i, st["bullet"]), leftIndent=12, bulletColor=ACCENT) for i in items],
        bulletType="bullet",
        start="circle",
        leftIndent=18,
        spaceBefore=2,
        spaceAfter=8,
    )


def cover(story, st, title: str, role: str, speaker: str, pages_hint: str):
    story.append(Spacer(1, 2.2 * cm))
    story.append(Paragraph("Project LAB — UNIMA Afya · Grupo 3", st["meta"]))
    story.append(Paragraph("On-Wallet / Carteira de Documentos", st["cover_sub"]))
    story.append(Spacer(1, 0.6 * cm))
    story.append(Paragraph(title, st["cover_title"]))
    story.append(Paragraph(role, st["cover_sub"]))
    story.append(Spacer(1, 0.8 * cm))
    box = Table(
        [[Paragraph(f"<b>Apresentador:</b> {speaker}", st["body"])],
         [Paragraph(f"<b>Roteiro de fala:</b> {pages_hint}", st["body"])],
         [Paragraph("<b>Producao:</b> https://carteira.eflow.space", st["body"])],
         [Paragraph("<b>Quadro Chronos:</b> Project LAB Repo (Grupo 3)", st["body"])]],
        colWidths=[15 * cm],
    )
    box.setStyle(TableStyle([
        ("BACKGROUND", (0, 0), (-1, -1), LIGHT),
        ("BOX", (0, 0), (-1, -1), 1, PRIMARY),
        ("LEFTPADDING", (0, 0), (-1, -1), 12),
        ("RIGHTPADDING", (0, 0), (-1, -1), 12),
        ("TOPPADDING", (0, 0), (-1, -1), 6),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 6),
    ]))
    story.append(box)
    story.append(Spacer(1, 1.2 * cm))
    story.append(Paragraph(
        "Este PDF e um roteiro individual: fale apenas sobre a sua parte. "
        "Nao precisa explicar o projeto inteiro — use o PDF geral do grupo para a visao completa.",
        st["body"],
    ))
    story.append(PageBreak())


def section(story, st, title: str):
    story.append(Paragraph(title, st["h1"]))


def subsection(story, st, title: str):
    story.append(Paragraph(title, st["h2"]))


def build_doc(path: Path, build_fn):
    st = styles()
    doc = SimpleDocTemplate(
        str(path),
        pagesize=A4,
        leftMargin=1.8 * cm,
        rightMargin=1.8 * cm,
        topMargin=1.6 * cm,
        bottomMargin=1.6 * cm,
        title=path.stem,
        author="Project LAB Grupo 3",
    )
    story: list = []
    build_fn(story, st)
    doc.build(story)
    print("wrote", path)


# --- PDFs ---

def pdf_geral(story, st):
    cover(
        story, st,
        "Relatorio Geral do Projeto",
        "Ementa academica + Quadro Chronos + Conformidade MVP",
        "Grupo 3 (todos)",
        "visao completa do produto",
    )
    section(story, st, "1. O que e o On-Wallet")
    story.append(Paragraph(
        "Carteira digital academica em que o titular guarda documentos (PDF/JPEG/PNG), "
        "registra integridade com SHA-256, gerencia ciclo de vida (versao, revogacao, expiracao) "
        "e compartilha links de verificacao com terceiros. Producao: carteira.eflow.space. "
        "Stack: Go + React + Postgres + MinIO, binario unico com SPA embutida e PM2.",
        st["body"],
    ))
    section(story, st, "2. Atores e casos de uso")
    story.append(Paragraph(
        "Documentados em docs/ATORES.md e docs/CASOS_DE_USO.md: Titular (sessao cookie), "
        "Terceiro verificador (token de share ou recibo /i/{id}) e Sistema (expiracao, isolamento). "
        "UC01–UC17 cobrem cadastro, upload, listagem, verify, recibo, versao, metadados, "
        "revogacao, exclusao, colecoes, shares e auditoria.",
        st["body"],
    ))
    section(story, st, "3. Requisitos funcionais RF-01 a RF-23")
    story.append(Paragraph(
        "Todos implementados (docs/REQUISITOS.md e docs/CONFORMIDADE.md). Resumo por modulo:",
        st["body"],
    ))
    bullets([
        "Acesso: RF-01/02 autenticacao e sessao HttpOnly",
        "Documentos: RF-03–07 upload, metadados, tipos/tags, colecoes, listagem/download",
        "Integridade: RF-08–12 hash, registro, verify, evidencias, recibo PDF+QR",
        "Ciclo de vida: RF-13–17 estados, versoes, revogacao, expiracao, PATCH metadados",
        "Compartilhamento: RF-18–20 links, prazo 1/7/30, validacao publica /v/:token",
        "Auditoria e limites: RF-21–23 audit, soft-delete, integridade ≠ autenticidade",
    ], st)
    section(story, st, "4. Regras de negocio RN-01 a RN-10")
    story.append(Paragraph(
        "docs/REGRAS_NEGOCIO.md — isolamento por titular, MIME/tamanho, hash na mesma TX, "
        "divergencia nao sobrescreve hash, integridade ≠ autenticidade, versoes imutaveis, "
        "revogacao com motivo, expiracao sem mudar hash, share com prazo, exclusao logica.",
        st["body"],
    ))
    section(story, st, "5. Arquitetura")
    bullets([
        "backend/internal/{auth,documents,shares,collections,assistant,config,db,storage}",
        "Padrao handler → service → repository",
        "Migrations 0001–0009 (users, verifications, shares, collections, assistant…)",
        "Frontend React: Layout autenticado + PublicChrome; visualizador PDF.js proprio",
        "Ops: Makefile (ci/build/deploy), ecosystem.config.js, health Postgres+MinIO",
    ], st)
    section(story, st, "6. Conformidade smoke (26/10/2026)")
    story.append(Paragraph(
        "Parecer docs/CONFORMIDADE.md: health ok; register 201; sem cookie 401; "
        "hash API = sha256sum; verify ok; cross-tenant 403; MIME 415; revogar sem motivo 422; "
        "share publico ok; recibo %PDF; soft-delete 204→404. Veredito: 23/23 RF e 10/10 RN.",
        st["body"],
    ))
    section(story, st, "7. Pos-MVP e fora de escopo")
    bullets([
        "Pos-MVP: assistente OCR (Tesseract) + Groq + RAG lexical + FAB; coerencia assistida (nao autenticidade)",
        "Visualizador PDF customizado (PDF.js) no lugar do iframe do Chrome",
        "Fora: OAuth Gov.br, autenticidade criptografica de orgao, DocTR sidecar, admin global",
    ], st)
    section(story, st, "8. Divisao do grupo (apresentacoes individuais)")
    bullets([
        "Irving Samuel — DevOps, APIs/integracoes, esqueleto backend, regras de negocio",
        "George — Funcionalidades backend, seguranca e integridade",
        "Joao Farias — Figma (DS), componentizacao FE, estilo e design patterns",
        "Paulo Paes — Componentizacao, QA e gestao (PM) no Chronos",
        "Rayssa — Estruturacao de telas e prototipagem de jornada (credito de equipe)",
    ], st)
    section(story, st, "9. Artefatos para citar na banca")
    for p in [
        "docs/REQUISITOS.md · REGRAS_NEGOCIO.md · CONFORMIDADE.md · ASSISTENTE.md",
        "docs/ATORES.md · CASOS_DE_USO.md · MODULOS.md · UML.md · FLUXOS.md · API.md",
        "Quadro Chronos: Project LAB Repo (Grupo 3) — cards RF-01…23 done",
        "Repositorio: github.com/IrvingSamuel/On-Wallet",
    ]:
        story.append(Paragraph(p, st["path"]))


def pdf_irving(story, st):
    cover(
        story, st,
        "Roteiro de Apresentacao",
        "DevOps · APIs e Integracoes · Esqueleto Backend · Regras de Negocio",
        "Irving Samuel",
        "6–8 minutos focados nesta parte",
    )
    section(story, st, "1. Sua mensagem central")
    story.append(Paragraph(
        "Voce entregou a base que permite o restante do time trabalhar: o esqueleto Go, "
        "contrato HTTP, ambiente de producao, integracoes Postgres/MinIO e a formalizacao "
        "das regras de negocio (RN) alinhadas a ementa e ao Chronos.",
        st["body"],
    ))
    section(story, st, "2. DevOps e operacao")
    bullets([
        "Dominio https://carteira.eflow.space — nginx → 127.0.0.1:7546",
        "PM2 processo carteira (ecosystem.config.js)",
        "Makefile: make install | frontend | backend | ci | build | deploy",
        "CI: tsc/vite + go vet + go build (-buildvcs=false)",
        "Health agregado: GET /api/v1/health (postgres + minio)",
        "Segredos apenas em .env (fora do git); modelo .env.example",
    ], st)
    subsection(story, st, "Evidencias")
    for p in ["Makefile", "ecosystem.config.js", "backend/cmd/server/main.go", "docs/CI.md", "docs/ARCHITECTURE.md"]:
        story.append(Paragraph(p, st["path"]))
    section(story, st, "3. Esqueleto backend e APIs")
    story.append(Paragraph(
        "O servidor ServeMux registra dominios auth, documents, shares, collections e assistant. "
        "A SPA React e embutida (backend/web). Config carrega PORT, DATABASE_URL, MinIO e Groq. "
        "docs/API.md documenta todos os endpoints com exemplos curl — use isso na fala.",
        st["body"],
    ))
    bullets([
        "Padrao handler → service → repository em internal/*",
        "Migrations versionadas em internal/db/migrations/",
        "Integracao MinIO (upload/download) e Postgres (metadados/integridade)",
        "Assistente: GROQ_API_KEY / modelos chat (pos-MVP orquestrado no boot)",
    ], st)
    for p in ["backend/internal/config/config.go", "docs/API.md", "backend/internal/httpx/httpx.go"]:
        story.append(Paragraph(p, st["path"]))
    section(story, st, "4. Regras de negocio")
    story.append(Paragraph(
        "docs/REGRAS_NEGOCIO.md formaliza RN-01…10 (isolamento, MIME, hash na TX, "
        "divergencia nao sobrescreve, integridade ≠ autenticidade, versoes imutaveis, "
        "revogacao com motivo, expiracao, share, soft-delete). Na banca, explique que "
        "as RNs amarram RF a comportamento observavel (status HTTP e UI).",
        st["body"],
    ))
    story.append(Paragraph("docs/REGRAS_NEGOCIO.md · docs/REQUISITOS.md · docs/CONFORMIDADE.md", st["path"]))
    section(story, st, "5. O que NAO precisa explicar")
    bullets([
        "Detalhe de cada query SQL de verify (George)",
        "Tokens CSS e componentes React (Joao)",
        "Checklist QA do Chronos (Paulo)",
        "Inventario de telas/jornada (Rayssa)",
    ], st)
    section(story, st, "6. Frases prontas")
    bullets([
        "O deploy e um binario Go com o frontend embutido; make deploy troca o binario no PM2.",
        "Sem sessao a API responde 401; documento de outro titular 403 — isolamento multi-tenant simples.",
        "As regras de negocio estao documentadas e batem com o smoke de conformidade.",
    ], st)


def pdf_george(story, st):
    cover(
        story, st,
        "Roteiro de Apresentacao",
        "Funcionalidades · Backend Completo · Seguranca · Integridade",
        "George",
        "6–8 minutos focados nesta parte",
    )
    section(story, st, "1. Sua mensagem central")
    story.append(Paragraph(
        "Voce fala do coracao do produto: como o arquivo entra, como o SHA-256 prova "
        "integridade, como verify/recibo/share/auditoria e soft-delete funcionam, "
        "e como a sessao protege o titular.",
        st["body"],
    ))
    section(story, st, "2. Dominios backend")
    bullets([
        "internal/auth — register/login/logout, cookie carteira_session, middleware",
        "internal/documents — upload, list, file, verify, receipt, versions, revoke, patch, delete",
        "internal/shares — create/list/revoke + GET public/shares/{token}",
        "internal/collections — CRUD e membership N:N",
        "internal/assistant — OCR Tesseract + coerencia Groq + chat RAG (pos-MVP)",
    ], st)
    for p in [
        "backend/internal/documents/service.go",
        "backend/internal/auth/handler.go",
        "backend/internal/shares/service.go",
        "backend/internal/db/migrations/",
    ]:
        story.append(Paragraph(p, st["path"]))
    section(story, st, "3. Integridade (RF-08…12)")
    bullets([
        "Hash SHA-256 dos bytes exatos no upload; integrity_records na mesma transacao",
        "POST /documents/{id}/verify recalcula MinIO; nunca sobrescreve hash (RN-04)",
        "GET /verifications — evidencia historica",
        "GET /receipt — PDF gofpdf + QR para /i/{id} (verificacao publica)",
        "Smoke: hash API = sha256sum local; verify → ok",
    ], st)
    section(story, st, "4. Seguranca e ciclo de vida")
    bullets([
        "Cookie HttpOnly; rotas privadas exigem sessao",
        "Cross-tenant → 403 forbidden",
        "MIME por magic bytes; .txt → 415",
        "Revogacao exige motivo (422 se vazio)",
        "Soft-delete deleted_at; MinIO/hash permanecem; shares invalidados",
        "Estados active / revoked / expired (+ enum replaced)",
    ], st)
    section(story, st, "5. RF-23 — o que voce deve enfatizar")
    story.append(Paragraph(
        "Integridade prova que o arquivo nao mudou. Autenticidade do emissor NÃO e verificada. "
        "A UI tem AuthNote e o box Autenticidade — Nao verificada. Coerencia assistida (OCR) "
        "nao e prova juridica de orgao.",
        st["body"],
    ))
    section(story, st, "6. O que NAO precisa explicar")
    bullets([
        "PM2/Makefile (Irving)",
        "Paleta CSS e PdfViewer (Joao)",
        "Processo PM/QA do quadro (Paulo)",
    ], st)


def pdf_joao(story, st):
    cover(
        story, st,
        "Roteiro de Apresentacao",
        "Figma (Design System) · Componentizacao FE · Estilo · Design Patterns",
        "Joao Farias",
        "6–8 minutos focados nesta parte",
    )
    section(story, st, "1. Sua mensagem central")
    story.append(Paragraph(
        "Voce apresenta a identidade visual e a camada de componentes React: do prototipo "
        "de Design System no Figma ate os tokens e componentes reutilizaveis no codigo, "
        "incluindo o visualizador PDF proprio alinhado a UI.",
        st["body"],
    ))
    section(story, st, "2. Design System (Figma → codigo)")
    story.append(Paragraph(
        "Card Chronos Design System (done): paleta primary #2966B1, accent green, faixa bandeira; "
        "tipografia Rawline/Raleway; focus-visible; alvos de toque. O Figma foi a referencia "
        "visual; a implementacao vive em styles.css e components/.",
        st["body"],
    ))
    bullets([
        "Tokens CSS em :root (cores, radius, sombra, fontes)",
        "StatusBadge, Hash+copiar, IntegrityResult, AuthNote, EmptyState",
        "Layout autenticado + PublicChrome (login, /v, /i)",
        "BrandLogo / identidade On-Wallet",
        "AssistantFab e PdfViewer no mesmo idioma visual",
    ], st)
    for p in [
        "frontend/src/styles.css",
        "frontend/src/components/ui.tsx",
        "frontend/src/components/Layout.tsx",
        "frontend/src/components/PdfViewer.tsx",
        "frontend/src/components/BrandLogo.tsx",
    ]:
        story.append(Paragraph(p, st["path"]))
    section(story, st, "3. Componentizacao e design patterns")
    bullets([
        "Componentes pequenos e reutilizaveis (um job por componente)",
        "DocumentPreview unifica imagem + PDF (lazy load do PDF.js)",
        "Espelho do padrao backend: UI consome api/client.ts tipado",
        "Estados visuais consistentes (ok / warn / bad / neutral)",
        "Acessibilidade: labels, confirmacoes destrutivas, :focus-visible",
    ], st)
    section(story, st, "4. Visualizador PDF proprio")
    story.append(Paragraph(
        "Substituimos o iframe do Chrome por PDF.js: toolbar de paginas, zoom e ajustar a largura, "
        "estilos da carteira. Mencione que isso melhora UX e controle de marca.",
        st["body"],
    ))
    section(story, st, "5. O que NAO precisa explicar")
    bullets([
        "Detalhe SHA-256/verify (George)",
        "Deploy PM2 (Irving)",
        "Checklist de QA e priorizacao de cards (Paulo)",
        "Mapa de jornada tela a tela (Rayssa)",
    ], st)


def pdf_paulo(story, st):
    cover(
        story, st,
        "Roteiro de Apresentacao",
        "Componentizacao · QA · Gestao de Projeto (PM)",
        "Paulo Paes",
        "6–8 minutos focados nesta parte",
    )
    section(story, st, "1. Sua mensagem central")
    story.append(Paragraph(
        "Voce fala de como o trabalho foi organizado no Chronos, como os criterios de aceite "
        "dos RF foram acompanhados, e como a componentizacao do frontend (estruturas de tela "
        "e reuso) foi validada com QA — sem entrar no detalhe visual do Figma (Joao).",
        st["body"],
    ))
    section(story, st, "2. PM no Chronos")
    bullets([
        "Quadro Project LAB Repo (Grupo 3) — cards RF-01…23 marcados done",
        "Status summaries nos cards como evidencia de entrega",
        "Priorizacao: base tecnica → integridade → ciclo de vida → shares → auth → docs academicos",
        "Acompanhamento de Design System e Frontend UI (cards com checklists)",
        "Entrega academica rastreavel: docs + conformidade + apresentacao",
    ], st)
    section(story, st, "3. QA — o que validar e como")
    story.append(Paragraph(
        "Use docs/CONFORMIDADE.md como script de QA. Na fala, destaque testes de aceite "
        "observaveis (status HTTP e UI), nao so 'funcionou no meu PC'.",
        st["body"],
    ))
    bullets([
        "Auth: register/login; rota privada sem cookie → 401",
        "Integridade: hash = sha256sum; verify ok; recibo PDF",
        "Seguranca: cross-tenant 403; MIME 415; revoke sem motivo 422",
        "Share publico /v/:token; recibo QR /i/:id sem login",
        "Soft-delete some da carteira; health 200 em producao",
        "RF-23: UI deixa claro autenticidade nao verificada",
    ], st)
    story.append(Paragraph("docs/CONFORMIDADE.md · docs/REQUISITOS.md · docs/API.md", st["path"]))
    section(story, st, "4. Componentizacao (visao de produto/QA)")
    bullets([
        "Validar que telas reutilizam StatusBadge, Hash, DocumentPreview, Layout",
        "Checar responsividade (drawer mobile) e estados vazios",
        "Garantir que mock banners nao aparecem em producao",
        "Card Chronos Frontend UI: 10 rotas cobertas",
    ], st)
    for p in ["frontend/src/App.tsx", "frontend/src/components/", "docs/CONTRIBUTING.md"]:
        story.append(Paragraph(p, st["path"]))
    section(story, st, "5. O que NAO precisa explicar")
    bullets([
        "Tokens CSS e Figma detalhado (Joao)",
        "Codigo Go de verify (George)",
        "Infra Makefile/PM2 (Irving)",
    ], st)


def pdf_rayssa(story, st):
    cover(
        story, st,
        "Roteiro de Apresentacao",
        "Estruturacao de Telas · Prototipagem de Jornada",
        "Rayssa",
        "6–8 minutos focados nesta parte",
    )
    section(story, st, "1. Sua mensagem central")
    story.append(Paragraph(
        "Voce apresenta a organizacao das telas e a jornada do usuario: o que cada rota "
        "existe para fazer, como o titular navega da carteira ao recibo/compartilhamento, "
        "e como o terceiro valida sem login. Foque em prototipagem e estruturacao de experiencia, "
        "nao em codigo de infraestrutura.",
        st["body"],
    ))
    section(story, st, "2. Inventario de telas (SPA)")
    bullets([
        "/login — autenticacao do titular",
        "/carteira — listagem, filtros, miniatura",
        "/documentos/novo — upload + metadados",
        "/documentos/:id — detalhe, acoes, integridade/coerencia/autenticidade",
        "/documentos/:id/recibo — recibo + QR + previa",
        "/documentos/:id/compartilhar — links e revogacao",
        "/verificar — verificacao de integridade (titular)",
        "/colecoes — agrupamentos",
        "/auditoria — trilha de eventos",
        "/v/:token e /i/:id — validacao publica (terceiro / recibo)",
    ], st)
    story.append(Paragraph("frontend/src/App.tsx · frontend/src/pages/*", st["path"]))
    section(story, st, "3. Jornadas para narrar")
    subsection(story, st, "Titular")
    story.append(Paragraph(
        "Login → Carteira → Novo documento → Detalhe (verificar / analisar IA) → "
        "Recibo (QR) ou Compartilhar (prazo) → Auditoria. Exclusao/revogacao com confirmacao.",
        st["body"],
    ))
    subsection(story, st, "Terceiro")
    story.append(Paragraph(
        "Recebe link /v/{token} ou escaneia QR do recibo (/i/{id}): ve previa + resultado "
        "de integridade, sem acessar a carteira completa. Textos deixam claro: nao e autenticidade.",
        st["body"],
    ))
    section(story, st, "4. Prototipagem e alinhamento visual")
    bullets([
        "Cada tela tem um proposito (page-head + acao primaria)",
        "Estados vazios e erros legiveis",
        "PublicChrome vs Layout autenticado (experiencia diferente para terceiro)",
        "Previa do documento presente nas telas de amostragem (detalhe, carteira, share, recibo, verify)",
        "FAB do assistente so no layout autenticado — nao polui a validacao publica",
    ], st)
    section(story, st, "5. O que NAO precisa explicar")
    bullets([
        "Implementacao SHA-256 (George)",
        "Deploy e Makefile (Irving)",
        "Tokens Figma/CSS profundos (Joao)",
        "Gestao de cards Chronos (Paulo)",
    ], st)
    section(story, st, "6. Frases prontas")
    bullets([
        "A estrutura de telas separa claramente titular e terceiro verificador.",
        "O fluxo da carteira conduz do upload ate a prova (recibo/share) sem misturar conceitos.",
        "A prototipagem priorizou amostragem do documento e clareza RF-23 em todas as superficies.",
    ], st)


def pdf_readme_apresentacao():
    text = """# PDFs de apresentacao — Grupo 3

Gerados por `scripts/gerar_apresentacao_pdfs.py`.

| Arquivo | Integrante / uso |
|---|---|
| 00-geral-grupo3-on-wallet.pdf | Visao completa (ementa + Chronos + conformidade) |
| 01-irving-devops-api-regras.pdf | Irving — DevOps, APIs, esqueleto, RNs |
| 02-george-backend-seguranca-integridade.pdf | George — Backend, seguranca, integridade |
| 03-joao-figma-frontend-design.pdf | Joao — Figma, FE, estilo, design patterns |
| 04-rayssa-telas-prototipagem.pdf | Rayssa — Telas e prototipagem (credito de equipe) |
| 05-paulo-componentizacao-qa-pm.pdf | Paulo — Componentizacao, QA, PM |

Regenerar:

```bash
python3 scripts/gerar_apresentacao_pdfs.py
```
"""
    (OUT / "README.md").write_text(text, encoding="utf-8")


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    build_doc(OUT / "00-geral-grupo3-on-wallet.pdf", pdf_geral)
    build_doc(OUT / "01-irving-devops-api-regras.pdf", pdf_irving)
    build_doc(OUT / "02-george-backend-seguranca-integridade.pdf", pdf_george)
    build_doc(OUT / "03-joao-figma-frontend-design.pdf", pdf_joao)
    build_doc(OUT / "04-rayssa-telas-prototipagem.pdf", pdf_rayssa)
    build_doc(OUT / "05-paulo-componentizacao-qa-pm.pdf", pdf_paulo)
    pdf_readme_apresentacao()
    print("OK", OUT)


if __name__ == "__main__":
    main()
