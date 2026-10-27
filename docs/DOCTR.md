# Parecer: uso do DocTR na carteira

**Card Chronos:** Verificar uso do DocTR  
**Fonte:** [mindee/doctr](https://github.com/mindee/doctr) · docs: https://mindee.github.io/doctr/  
**Data:** 26/10/2026  
**Decisão MVP:** **não integrar DocTR** (sidecar Python/PyTorch).  
**Reabertura (out/2026):** OCR assistido via **Groq vision** + extração de texto PDF nativa — ver [ASSISTENTE.md](./ASSISTENTE.md). Continua **sem** DocTR.

---

## 1. O que é o DocTR

docTR (Document Text Recognition) é uma biblioteca **Python** de OCR / compreensão de documentos (Mindee, Apache-2.0):

- Pipeline típico: **detecção** de regiões de texto + **reconhecimento** (OCR de palavras/linhas)
- Desde v1.x: também layout, tabelas e export Markdown/HTML (toolkit mais amplo)
- Backend de ML: **PyTorch** (TensorFlow foi removido)
- Instalação: `pip install python-doctr` (Python **≥ 3.11**)
- Entrada: PDF e imagens (`DocumentFile.from_pdf` / `from_images`)
- Uso mínimo:

```python
from doctr.io import DocumentFile
from doctr.models import ocr_predictor

doc = DocumentFile.from_pdf("doc.pdf")
model = ocr_predictor(pretrained=True)  # .cuda() se houver GPU
result = model(doc)
print(result.render())
```

Inferência em CPU é viável mas lenta; GPU (CUDA/MPS) é o cenário realista para UX aceitável.

---

## 2. Fit com a carteira atual

A carteira (`carteira.eflow.space`) é um binário **Go** + React + Postgres + MinIO. O MVP (RF-01…23) já entrega:

| Capacidade | Como é feito hoje | Precisa de OCR? |
|---|---|---|
| Guardar PDF/JPEG/PNG | `POST /documents` → MinIO | Não |
| Prova de **integridade** | SHA-256 + `integrity_records` + verify | **Não** |
| Distinguir autenticidade | UI explicita (RF-23): autenticidade não verificada | Não (OCR também **não** prova autenticidade do emissor) |
| Metadados (título, tipo, tags) | Informados pelo titular no upload | OCR só ajudaria a **sugerir** texto |

O nome do quadro (“verificação de imagem”) no contexto do projeto significa **verificação de integridade do arquivo**, não OCR/visão computacional. Confundir os dois termos é um risco de escopo.

**Único encaixe útil do DocTR (pós-MVP):** após o upload, extrair texto e sugerir `title` / tags — feature de conveniência, não requisito de integridade.

---

## 3. Custos de integração

Integrar DocTR obrigaria sair do monolito Go atual (`backend/cmd/server`):

| Item | Impacto |
|---|---|
| Runtime | Sidecar Python + PyTorch (centenas de MB / GB de modelos) |
| Infra | GPU desejável; sem GPU, fila assíncrona + latência alta no upload |
| API | Novo serviço HTTP interno; job após Put no MinIO; status “processando OCR” |
| Ops | Imagem Docker separada, health, versionamento de pesos |
| Esforço | ≈ **6,5 dias-pessoa PERT** ([ESTIMATIVA.md](./ESTIMATIVA.md) §3.3) |

Isso conflita com a arquitetura simples já em produção e com o prazo acadêmico restante (docs UML/atores/RN).

---

## 4. Alternativas (se OCR for exigido depois)

| Opção | Quando considerar |
|---|---|
| **OnnxTR** (modelos ONNX do ecossistema docTR, sem PyTorch completo) | Precisar OCR local mais leve |
| API OCR externa (cloud) | Demo rápida sem operar GPU |
| Manter metadados manuais | Default do MVP — já atende RF-03/04/05 |

Nenhuma dessas opções é necessária para fechar RF-01…23.

---

## 5. Decisão e critérios de reabertura

**DocTR:** permanece fora — custo de ops incompatível com o monolito Go.

**Alternativa adotada:** assistente Groq (OCR/visão + chat + RAG lexical) para **coerência assistida**, sem afirmar autenticidade. Detalhes em [ASSISTENTE.md](./ASSISTENTE.md).
