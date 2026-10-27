package documents

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/phpdave11/gofpdf"
	qrcode "github.com/skip2/go-qrcode"
)

// ReceiptPDF builds an integrity receipt PDF for the document's current version (RF-12).
func (s *Service) ReceiptPDF(ctx context.Context, ownerID string, docID uuid.UUID) ([]byte, string, error) {
	detail, err := s.repo.GetDetail(ctx, ownerID, docID)
	if err != nil {
		return nil, "", err
	}
	if detail.Integrity == nil || detail.Integrity.Hash == "" {
		return nil, "", ErrMissingIntegrity
	}

	verifyURL := fmt.Sprintf("%s/i/%s", s.publicOrigin, detail.Document.ID.String())
	qrPNG, err := qrcode.Encode(verifyURL, qrcode.Medium, 256)
	if err != nil {
		return nil, "", fmt.Errorf("qrcode: %w", err)
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Recibo de integridade", false)
	pdf.AddPage()
	pdf.SetMargins(20, 20, 20)

	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 10, "Recibo de verificacao de integridade", "", 1, "L", false, 0, "")
	pdf.Ln(4)

	pdf.SetFont("Arial", "", 11)
	writeKV(pdf, "Documento", detail.Document.Title)
	writeKV(pdf, "ID do documento", detail.Document.ID.String())
	writeKV(pdf, "Versao", fmt.Sprintf("v%d", detail.Integrity.Version))
	writeKV(pdf, "Algoritmo", detail.Integrity.Algorithm)
	writeKV(pdf, "Data/hora do registro", detail.Integrity.CreatedAt.UTC().Format(time.RFC3339))
	writeKV(pdf, "Hash", detail.Integrity.Hash)
	writeKV(pdf, "URL de verificacao", verifyURL)

	pdf.Ln(6)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 8, "Politica de verificacao", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.MultiCell(0, 5, "O hash SHA-256 foi calculado sobre os bytes exatos do arquivo no momento do envio. Qualquer alteracao no conteudo produz um hash diferente. Este recibo comprova integridade; nao comprova autenticidade do emissor.", "", "L", false)

	pdf.Ln(8)
	opt := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
	pdf.RegisterImageOptionsReader("qr", opt, bytes.NewReader(qrPNG))
	pdf.ImageOptions("qr", 20, pdf.GetY(), 40, 40, false, opt, 0, "")
	pdf.SetY(pdf.GetY() + 42)
	pdf.SetFont("Arial", "I", 9)
	pdf.CellFormat(0, 5, "Escaneie o QR para abrir a tela de verificacao", "", 1, "L", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, "", fmt.Errorf("pdf output: %w", err)
	}
	filename := fmt.Sprintf("recibo-%s-v%d.pdf", detail.Document.ID.String(), detail.Integrity.Version)
	return buf.Bytes(), filename, nil
}

func writeKV(pdf *gofpdf.Fpdf, label, value string) {
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(45, 6, label, "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.MultiCell(0, 6, value, "", "L", false)
}
