import { useEffect, useRef, useState } from "react";
import * as pdfjs from "pdfjs-dist";
import type { PDFDocumentProxy } from "pdfjs-dist";
import pdfWorker from "pdfjs-dist/build/pdf.worker.min.mjs?url";

pdfjs.GlobalWorkerOptions.workerSrc = pdfWorker;

type Props = {
  src: string;
  title?: string;
  compact?: boolean;
};

const ZOOM_STEPS = [0.75, 1, 1.25, 1.5, 2];

/** Custom in-app PDF viewer (canvas + toolbar) — no browser/Chrome PDF plugin. */
export function PdfViewer({ src, title = "Documento", compact }: Props) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const wrapRef = useRef<HTMLDivElement>(null);
  const docRef = useRef<PDFDocumentProxy | null>(null);

  const [page, setPage] = useState(1);
  const [pageCount, setPageCount] = useState(0);
  const [zoomIdx, setZoomIdx] = useState(1);
  const [fitWidth, setFitWidth] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [rendering, setRendering] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");
    setPage(1);
    setPageCount(0);
    docRef.current = null;

    const task = pdfjs.getDocument({
      url: src,
      withCredentials: true,
    });

    task.promise
      .then((doc) => {
        if (cancelled) {
          void doc.destroy();
          return;
        }
        docRef.current = doc;
        setPageCount(doc.numPages);
        setLoading(false);
      })
      .catch((e) => {
        if (cancelled) return;
        setError(e instanceof Error ? e.message : "Falha ao carregar o PDF.");
        setLoading(false);
      });

    return () => {
      cancelled = true;
      void task.destroy();
      if (docRef.current) {
        void docRef.current.destroy();
        docRef.current = null;
      }
    };
  }, [src]);

  useEffect(() => {
    const doc = docRef.current;
    const canvas = canvasRef.current;
    if (!doc || !canvas || loading || error || page < 1 || page > pageCount) return;

    let cancelled = false;
    setRendering(true);

    (async () => {
      try {
        const pdfPage = await doc.getPage(page);
        if (cancelled) return;

        const base = pdfPage.getViewport({ scale: 1 });
        const wrapW = wrapRef.current?.clientWidth ?? base.width;
        const scale = fitWidth
          ? Math.max(0.5, (wrapW - 24) / base.width)
          : ZOOM_STEPS[zoomIdx] ?? 1;
        const viewport = pdfPage.getViewport({ scale });

        const ctx = canvas.getContext("2d");
        if (!ctx) return;

        const outputScale = window.devicePixelRatio || 1;
        canvas.width = Math.floor(viewport.width * outputScale);
        canvas.height = Math.floor(viewport.height * outputScale);
        canvas.style.width = `${Math.floor(viewport.width)}px`;
        canvas.style.height = `${Math.floor(viewport.height)}px`;
        ctx.setTransform(outputScale, 0, 0, outputScale, 0, 0);

        await pdfPage.render({ canvasContext: ctx, viewport }).promise;
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : "Falha ao renderizar a página.");
        }
      } finally {
        if (!cancelled) setRendering(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [page, pageCount, zoomIdx, fitWidth, loading, error, src]);

  if (error) {
    return (
      <div className="pdf-viewer pdf-viewer--error" role="alert">
        <p className="muted">Não foi possível exibir o PDF no visualizador.</p>
        <a className="btn ghost sm" href={src} target="_blank" rel="noreferrer">
          Baixar / abrir arquivo
        </a>
      </div>
    );
  }

  return (
    <div className={`pdf-viewer ${compact ? "pdf-viewer--compact" : ""}`} aria-label={`PDF: ${title}`}>
      {!compact && (
        <div className="pdf-toolbar" role="toolbar" aria-label="Controles do PDF">
          <div className="pdf-toolbar-group">
            <button
              type="button"
              className="btn ghost sm"
              disabled={loading || page <= 1}
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              aria-label="Página anterior"
            >
              ←
            </button>
            <span className="pdf-page-label" aria-live="polite">
              {loading ? "…" : `${page} / ${pageCount || "—"}`}
            </span>
            <button
              type="button"
              className="btn ghost sm"
              disabled={loading || page >= pageCount}
              onClick={() => setPage((p) => Math.min(pageCount, p + 1))}
              aria-label="Próxima página"
            >
              →
            </button>
          </div>
          <div className="pdf-toolbar-group">
            <button
              type="button"
              className={`btn ghost sm ${fitWidth ? "pdf-tool-active" : ""}`}
              disabled={loading}
              onClick={() => setFitWidth(true)}
              title="Ajustar à largura"
            >
              Largura
            </button>
            <button
              type="button"
              className="btn ghost sm"
              disabled={loading || fitWidth || zoomIdx <= 0}
              onClick={() => {
                setFitWidth(false);
                setZoomIdx((i) => Math.max(0, i - 1));
              }}
              aria-label="Diminuir zoom"
            >
              −
            </button>
            <button
              type="button"
              className={`btn ghost sm ${!fitWidth ? "pdf-tool-active" : ""}`}
              disabled={loading}
              onClick={() => {
                setFitWidth(false);
                setZoomIdx(1);
              }}
              title="Zoom 100%"
            >
              {fitWidth ? "Auto" : `${Math.round((ZOOM_STEPS[zoomIdx] ?? 1) * 100)}%`}
            </button>
            <button
              type="button"
              className="btn ghost sm"
              disabled={loading || zoomIdx >= ZOOM_STEPS.length - 1}
              onClick={() => {
                setFitWidth(false);
                setZoomIdx((i) => Math.min(ZOOM_STEPS.length - 1, i + 1));
              }}
              aria-label="Aumentar zoom"
            >
              +
            </button>
          </div>
        </div>
      )}

      <div className="pdf-stage" ref={wrapRef}>
        {loading && <p className="muted pdf-status">Carregando PDF…</p>}
        {!loading && rendering && <p className="muted pdf-status pdf-status--overlay">Renderizando…</p>}
        <canvas ref={canvasRef} className="pdf-canvas" hidden={loading} />
      </div>
    </div>
  );
}
