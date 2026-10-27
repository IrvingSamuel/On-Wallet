import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { QRCodeSVG } from "qrcode.react";
import { ApiError, api, type DocumentDetail } from "../api/client";
import { DocumentPreview } from "../components/DocumentPreview";
import { Hash, formatDate } from "../components/ui";

export default function Receipt() {
  const { id = "" } = useParams();
  const [detail, setDetail] = useState<DocumentDetail | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    api
      .getDocument(id)
      .then((d) => {
        if (!cancelled) setDetail(d);
      })
      .catch((e) => {
        if (!cancelled) {
          setError(
            e instanceof ApiError && (e.status === 404 || e.status === 400)
              ? "Documento não encontrado para este titular."
              : "Falha ao carregar o recibo.",
          );
        }
      });
    return () => {
      cancelled = true;
    };
  }, [id]);

  const title = detail?.document.title ?? "Documento";
  const integrity = detail?.integrity;
  const algorithm = integrity?.algorithm ?? "SHA-256";
  const verifyUrl = `${typeof location !== "undefined" ? location.origin : ""}/i/${id}`;

  return (
    <>
      <div className="page-head">
        <div>
          <Link to={`/documentos/${id}`} className="muted">← {title}</Link>
          <h1>Recibo de integridade</h1>
        </div>
        <a
          className="btn primary"
          href={detail ? api.receiptUrl(id) : undefined}
          download={detail ? `recibo-${id}-v${detail.integrity?.version ?? 1}.pdf` : undefined}
          aria-disabled={!detail || !detail.integrity}
          onClick={(e) => {
            if (!detail?.integrity) e.preventDefault();
          }}
        >
          Baixar PDF
        </a>
      </div>
      {error && <div className="alert revoked">{error}</div>}

      {detail && (
        <div className="card">
          <h3>Pré-visualização</h3>
          <DocumentPreview
            src={api.downloadUrl(id)}
            mimeType={detail.version.mime_type}
            title={detail.document.title}
          />
        </div>
      )}

      <div className="receipt">
        <div className="receipt-flag" aria-hidden="true" />
        <div className="receipt-body">
          <h2>Recibo de verificação de integridade</h2>
          <p className="muted">Documento: <strong>{title}</strong></p>
          <dl className="kv">
            <dt>ID do documento</dt>
            <dd><code>{id}</code></dd>
            <dt>Algoritmo</dt>
            <dd>{algorithm}</dd>
            <dt>Versão</dt>
            <dd>{integrity ? `v${integrity.version}` : "—"}</dd>
            <dt>Data/hora do registro</dt>
            <dd>{integrity ? formatDate(integrity.created_at) : "—"}</dd>
            <dt>Hash</dt>
            <dd>{integrity ? <Hash value={integrity.hash} /> : "—"}</dd>
          </dl>
          <div>
            <h3>Política de verificação</h3>
            <p className="muted" style={{ fontSize: "0.875rem" }}>
              O hash SHA-256 foi calculado sobre os bytes exatos do arquivo no momento do envio.
              Qualquer alteração no conteúdo produz um hash diferente. Este recibo comprova
              integridade; não comprova autenticidade do emissor.
            </p>
          </div>
          <div className="receipt-qr">
            <QRCodeSVG value={verifyUrl} size={140} />
          </div>
          <small className="muted" style={{ textAlign: "center" }}>
            Escaneie para abrir a tela de verificação
          </small>
        </div>
      </div>
    </>
  );
}
