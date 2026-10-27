package assistant

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/ledongthuc/pdf"
)

const minUsefulText = 40

type OCRResult struct {
	Text   string
	Method string // pdf_text | tesseract | pdf_hybrid | vision | unreadable
}

func (s *Service) extractText(ctx context.Context, mime string, data []byte) (OCRResult, error) {
	switch mime {
	case "application/pdf":
		return s.extractPDF(ctx, data)
	case "image/jpeg", "image/png":
		return s.ocrImage(ctx, mime, data)
	default:
		return OCRResult{Method: "unreadable"}, nil
	}
}

// extractPDF merges the PDF text layer with Tesseract on rasterized pages.
// CNH digital / docs Serpro often expose only a signature cover in the text layer;
// the real fields live in the rendered image.
func (s *Service) extractPDF(ctx context.Context, data []byte) (OCRResult, error) {
	textLayer, _ := extractPDFText(data)
	rasterOCR, _ := s.ocrPDFPages(ctx, data, 3)

	combined := mergeOCRParts(textLayer, rasterOCR)
	if utf8.RuneCountInString(strings.TrimSpace(combined)) < minUsefulText {
		if strings.TrimSpace(textLayer) != "" {
			return OCRResult{Text: cleanText(textLayer), Method: "pdf_text"}, nil
		}
		return OCRResult{Method: "unreadable"}, nil
	}

	method := "pdf_hybrid"
	switch {
	case strings.TrimSpace(rasterOCR) == "":
		method = "pdf_text"
	case strings.TrimSpace(textLayer) == "":
		method = "tesseract"
	}
	return OCRResult{Text: cleanText(combined), Method: method}, nil
}

func mergeOCRParts(parts ...string) string {
	var b strings.Builder
	seen := map[string]bool{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(p)
	}
	return b.String()
}

func (s *Service) ocrPDFPages(ctx context.Context, data []byte, maxPages int) (string, error) {
	if maxPages <= 0 {
		maxPages = 3
	}
	bin, err := exec.LookPath("pdftoppm")
	if err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp("", "carteira-ocr-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)

	in := filepath.Join(dir, "in.pdf")
	if err := os.WriteFile(in, data, 0o600); err != nil {
		return "", err
	}
	outPrefix := filepath.Join(dir, "page")
	// Higher DPI helps dense CNH digital layouts.
	cmd := exec.CommandContext(ctx, bin, "-png", "-f", "1", "-l", fmt.Sprintf("%d", maxPages), "-r", "200", in, outPrefix)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("pdftoppm: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	var b strings.Builder
	for i := 1; i <= maxPages; i++ {
		pngPath := fmt.Sprintf("%s-%d.png", outPrefix, i)
		img, err := os.ReadFile(pngPath)
		if err != nil {
			continue
		}
		txt, err := tesseractOCR(ctx, "image/png", img)
		if err != nil || strings.TrimSpace(txt) == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "--- página %d ---\n%s", i, strings.TrimSpace(txt))
	}
	return b.String(), nil
}

func (s *Service) ocrImage(ctx context.Context, mime string, data []byte) (OCRResult, error) {
	if out, err := tesseractOCR(ctx, mime, data); err == nil && utf8.RuneCountInString(strings.TrimSpace(out)) >= 8 {
		return OCRResult{Text: cleanText(out), Method: "tesseract"}, nil
	}
	if s.groq != nil && s.groq.Enabled() && s.groq.visionModel != "" {
		res, err := s.visionOCR(ctx, mime, data)
		if err == nil && res.Method != "unreadable" {
			return res, nil
		}
	}
	return OCRResult{Method: "unreadable"}, nil
}

func (s *Service) visionOCR(ctx context.Context, mime string, data []byte) (OCRResult, error) {
	if s.groq == nil || !s.groq.Enabled() {
		return OCRResult{Method: "unreadable"}, ErrUnavailable
	}
	prompt := `Extraia TODO o texto legível deste documento brasileiro (RG, CPF, CNH, certidão, diploma, comprovante etc.).
Responda só com o texto OCR, linha a linha. Se ilegível, responda exatamente: UNREADABLE.`
	out, err := s.groq.VisionOCR(ctx, mime, data, prompt)
	if err != nil {
		return OCRResult{}, err
	}
	if strings.EqualFold(strings.TrimSpace(out), "UNREADABLE") || strings.TrimSpace(out) == "" {
		return OCRResult{Method: "unreadable"}, nil
	}
	return OCRResult{Text: cleanText(out), Method: "vision"}, nil
}

func tesseractOCR(ctx context.Context, mime string, data []byte) (string, error) {
	bin, err := exec.LookPath("tesseract")
	if err != nil {
		return "", fmt.Errorf("tesseract not found")
	}
	ext := ".jpg"
	if mime == "image/png" {
		ext = ".png"
	}
	dir, err := os.MkdirTemp("", "carteira-tess-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	in := filepath.Join(dir, "in"+ext)
	if err := os.WriteFile(in, data, 0o600); err != nil {
		return "", err
	}
	outBase := filepath.Join(dir, "out")
	// PSM 4: assume a single column of variable-size text (better for CNH cards).
	cmd := exec.CommandContext(ctx, bin, in, outBase, "-l", "por+eng", "--psm", "4")
	if b, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("tesseract: %w (%s)", err, strings.TrimSpace(string(b)))
	}
	text, err := os.ReadFile(outBase + ".txt")
	if err != nil {
		return "", err
	}
	return string(text), nil
}

func extractPDFText(data []byte) (string, error) {
	if bin, err := exec.LookPath("pdftotext"); err == nil {
		cmd := exec.Command(bin, "-layout", "-q", "-", "-")
		cmd.Stdin = bytes.NewReader(data)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err == nil {
			return stdout.String(), nil
		}
	}
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	var b strings.Builder
	n := r.NumPage()
	if n > 8 {
		n = 8
	}
	for i := 1; i <= n; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		txt, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		b.WriteString(txt)
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func cleanText(s string) string {
	s = strings.ReplaceAll(s, "\x00", "")
	s = strings.TrimSpace(s)
	if len(s) > 20000 {
		s = s[:20000]
	}
	return s
}
