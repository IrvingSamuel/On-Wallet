package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type fieldExtract struct {
	TitleHint  string   `json:"title_hint"`
	DocType    string   `json:"doc_type"`
	OriginHint string   `json:"origin_hint"`
	Names      []string `json:"names"`
	IDs        []string `json:"ids"`
	Dates      []string `json:"dates"`
	Notes      string   `json:"notes"`
}

type consistencyLLM struct {
	Result  string          `json:"result"`
	Summary string          `json:"summary"`
	Fields  json.RawMessage `json:"fields"`
	Details json.RawMessage `json:"details"`
}

func (s *Service) analyzeConsistency(ctx context.Context, meta docVersionRow, ocrText string) (ConsistencyCheck, json.RawMessage, error) {
	if strings.TrimSpace(ocrText) == "" {
		details, _ := json.Marshal(map[string]string{"reason": "empty_ocr"})
		return ConsistencyCheck{
			Result:  ResultUnreadable,
			Summary: "Não foi possível ler texto suficiente do arquivo para comparar com os metadados.",
			Details: details,
		}, json.RawMessage(`{}`), nil
	}
	if s.groq == nil || !s.groq.Enabled() {
		return ConsistencyCheck{}, nil, ErrUnavailable
	}

	system := `Você auxilia a carteira digital acadêmica. Compare o texto OCR de um documento com os metadados declarados pelo titular.
NÃO afirme autenticidade do emissor. NÃO diga que o documento é legítimo ou forjado.
Responda APENAS JSON válido com este schema:
{
  "result": "consistent" | "divergent" | "unreadable",
  "summary": "frase curta em português",
  "fields": {
    "title_hint": "",
    "doc_type": "rg|cpf|cnh|certidao|certificado|diploma|comprovante|outro|",
    "origin_hint": "",
    "names": [],
    "ids": [],
    "dates": [],
    "notes": ""
  },
  "details": { "matches": [], "mismatches": [] }
}
Regras:
- "consistent" se título/tipo/origem forem razoavelmente compatíveis com o OCR (não exige campos perfeitos).
- CNH digital: menções a CARTEIRA NACIONAL DE HABILITAÇÃO, DRIVER LICENSE, SENATRAN, SERPRO, Ministério dos Transportes, MRZ (I<BRA…), UF (ex. ALAGOAS) ou capa Assinador Serpro + dados de habilitação → compatível com type=cnh / origem SSP-*/DETRAN/SENATRAN.
- Não marque divergent só porque a capa Serpro aparece; use o conjunto do OCR.
- "divergent" só com contradição clara (ex.: metadado CNH e OCR é boleto/nota fiscal).
- "unreadable" se o OCR for insuficiente.`

	user := fmt.Sprintf(`Metadados declarados:
- title: %s
- type: %s
- origin: %s
- tags: %s
- file_name: %s

Texto OCR:
%s`, meta.Title, meta.Type, meta.Origin, strings.Join(meta.Tags, ", "), meta.FileName, ocrText)

	raw, err := s.groq.Chat(ctx, system, []ChatMessage{{Role: "user", Content: user}})
	if err != nil {
		return ConsistencyCheck{}, nil, err
	}
	raw = stripJSONFence(raw)
	var parsed consistencyLLM
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		details, _ := json.Marshal(map[string]string{"parse_error": err.Error(), "raw": truncate(raw, 500)})
		return ConsistencyCheck{
			Result:  ResultError,
			Summary: "Falha ao interpretar a resposta do modelo de coerência.",
			Details: details,
		}, json.RawMessage(`{}`), nil
	}
	result := parsed.Result
	switch result {
	case ResultConsistent, ResultDivergent, ResultUnreadable:
	default:
		result = ResultError
	}
	fields := parsed.Fields
	if len(fields) == 0 {
		fields = json.RawMessage(`{}`)
	}
	details := parsed.Details
	if len(details) == 0 {
		details = json.RawMessage(`{}`)
	}
	summary := strings.TrimSpace(parsed.Summary)
	if summary == "" {
		summary = "Análise de coerência concluída."
	}
	return ConsistencyCheck{Result: result, Summary: summary, Details: details}, fields, nil
}

func stripJSONFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```JSON")
		s = strings.TrimPrefix(s, "```")
		if i := strings.LastIndex(s, "```"); i >= 0 {
			s = s[:i]
		}
	}
	return strings.TrimSpace(s)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
