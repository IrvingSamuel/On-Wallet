import { lazy, Suspense, useState } from "react";

const PdfViewer = lazy(() =>
  import("./PdfViewer").then((m) => ({ default: m.PdfViewer })),
);

type Props = {
  src: string;
  mimeType: string;
  title?: string;
  /** Compact thumbnail for cards/lists */
  compact?: boolean;
};

function kind(mime: string): "image" | "pdf" | "other" {
  const m = mime.toLowerCase();
  if (m.startsWith("image/")) return "image";
  if (m === "application/pdf" || m.includes("pdf")) return "pdf";
  return "other";
}

function downloadHref(src: string): string {
  return `${src}${src.includes("?") ? "&" : "?"}download=1`;
}

/** Inline preview of a stored document (PDF / JPEG / PNG). */
export function DocumentPreview({ src, mimeType, title = "Documento", compact }: Props) {
  const [failed, setFailed] = useState(false);
  const k = kind(mimeType);

  if (failed || k === "other") {
    return (
      <div className={`doc-preview ${compact ? "doc-preview--compact" : ""} doc-preview--empty`} role="status">
        <p className="muted">Pré-visualização indisponível{mimeType ? ` (${mimeType})` : ""}.</p>
        {!compact && (
          <a className="btn ghost sm" href={downloadHref(src)} download>
            Baixar arquivo
          </a>
        )}
      </div>
    );
  }

  if (k === "pdf") {
    return (
      <div className={`doc-preview ${compact ? "doc-preview--compact" : ""}`}>
        <div className={`doc-preview-frame doc-preview-frame--pdf ${compact ? "doc-preview-frame--pdf-compact" : ""}`}>
          <Suspense
            fallback={
              <div className="pdf-viewer">
                <p className="muted pdf-status">Carregando visualizador…</p>
              </div>
            }
          >
            <PdfViewer src={src} title={title} compact={compact} />
          </Suspense>
        </div>
        {!compact && (
          <div className="doc-preview-actions">
            <a className="btn ghost sm" href={downloadHref(src)} download>
              Baixar PDF
            </a>
          </div>
        )}
      </div>
    );
  }

  return (
    <div className={`doc-preview ${compact ? "doc-preview--compact" : ""}`}>
      <div className="doc-preview-frame">
        <img
          src={src}
          alt={`Pré-visualização: ${title}`}
          onError={() => setFailed(true)}
        />
      </div>
      {!compact && (
        <div className="doc-preview-actions">
          <a className="btn ghost sm" href={src} target="_blank" rel="noreferrer">
            Abrir imagem
          </a>
        </div>
      )}
    </div>
  );
}
