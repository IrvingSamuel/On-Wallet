import { useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { QRCodeSVG } from "qrcode.react";
import { ApiError, api, type ShareLink } from "../api/client";
import { DocumentPreview } from "../components/DocumentPreview";
import { formatDate } from "../components/ui";

function remaining(expiresAt: string) {
  const ms = new Date(expiresAt).getTime() - Date.now();
  if (ms <= 0) return "Expirado";
  const days = Math.floor(ms / 864e5);
  const hours = Math.floor((ms % 864e5) / 36e5);
  if (days > 0) return `${days}d ${hours}h restantes`;
  return `${hours}h restantes`;
}

export default function Share() {
  const { id = "" } = useParams();
  const [title, setTitle] = useState("Documento");
  const [mimeType, setMimeType] = useState("");
  const [days, setDays] = useState(7);
  const [links, setLinks] = useState<ShareLink[]>([]);
  const [copied, setCopied] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  function loadShares() {
    return api.listShares(id).then((r) => setLinks(r.items));
  }

  useEffect(() => {
    let cancelled = false;
    setError("");
    Promise.all([api.getDocument(id), api.listShares(id)])
      .then(([d, shares]) => {
        if (cancelled) return;
        setTitle(d.document.title);
        setMimeType(d.version.mime_type);
        setLinks(shares.items);
      })
      .catch((e) => {
        if (cancelled) return;
        if (e instanceof ApiError && (e.status === 404 || e.status === 400)) {
          setTitle("Documento");
          setError("Documento não encontrado.");
        } else {
          setError(e instanceof ApiError ? e.message : "Falha ao carregar compartilhamentos.");
        }
      });
    return () => {
      cancelled = true;
    };
  }, [id]);

  const latest = useMemo(
    () => links.find((l) => !l.revoked && new Date(l.expires_at).getTime() > Date.now()),
    [links],
  );
  const url = latest ? `${location.origin}/v/${latest.token}` : "";

  async function generate() {
    setBusy(true);
    setError("");
    try {
      await api.createShare(id, days);
      await loadShares();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Falha ao gerar link.");
    } finally {
      setBusy(false);
    }
  }

  async function revoke(shareId: string) {
    setBusy(true);
    setError("");
    try {
      await api.revokeShare(shareId);
      await loadShares();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Falha ao revogar link.");
    } finally {
      setBusy(false);
    }
  }

  async function copy() {
    if (!url) return;
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* ignore */
    }
  }

  return (
    <>
      <div className="page-head">
        <div>
          <Link to={`/documentos/${id}`} className="muted">← {title}</Link>
          <h1>Compartilhar documento</h1>
        </div>
      </div>
      <p className="muted">O terceiro verá apenas este documento e o resultado da verificação de integridade.</p>
      {error && <div className="alert revoked">{error}</div>}

      {mimeType && (
        <div className="card">
          <h3>Pré-visualização</h3>
          <DocumentPreview src={api.downloadUrl(id)} mimeType={mimeType} title={title} />
        </div>
      )}

      <div className="detail-grid">
        <div className="card">
          <h3>Novo link</h3>
          <label>
            Validade
            <select value={days} onChange={(e) => setDays(Number(e.target.value))} disabled={busy}>
              <option value={1}>1 dia</option>
              <option value={7}>7 dias</option>
              <option value={30}>30 dias</option>
            </select>
          </label>
          <button type="button" className="btn primary" onClick={generate} disabled={busy}>
            {busy ? "Gerando…" : "Gerar link"}
          </button>
        </div>

        <div className="card center">
          {latest ? (
            <>
              <QRCodeSVG value={url} size={180} />
              <code className="share-url">{url}</code>
              <small className="muted">Expira em {formatDate(latest.expires_at)}</small>
              <div className="actions">
                <button type="button" className="btn ghost" onClick={copy}>
                  {copied ? "Copiado" : "Copiar link"}
                </button>
                <a className="btn ghost" href={url} target="_blank" rel="noreferrer">Abrir como terceiro</a>
              </div>
            </>
          ) : (
            <p className="muted">Nenhum link ativo. Gere um novo.</p>
          )}
        </div>
      </div>

      <div className="card">
        <h3>Links</h3>
        {links.length === 0 ? (
          <p className="muted">Nenhum link gerado ainda.</p>
        ) : (
          <ul style={{ listStyle: "none", padding: 0, margin: 0, display: "flex", flexDirection: "column", gap: "0.75rem" }}>
            {links.map((l) => (
              <li key={l.id} style={{ display: "flex", flexWrap: "wrap", gap: "0.5rem", alignItems: "center", justifyContent: "space-between" }}>
                <div>
                  <code style={{ fontSize: "0.8rem", wordBreak: "break-all" }}>{l.token.slice(0, 16)}…</code>
                  <div className="muted" style={{ fontSize: "0.8125rem" }}>
                    {l.revoked ? "Revogado" : remaining(l.expires_at)} · criado em {formatDate(l.created_at)}
                  </div>
                </div>
                {!l.revoked && new Date(l.expires_at).getTime() > Date.now() && (
                  <button
                    type="button"
                    className="btn danger sm"
                    disabled={busy}
                    onClick={() => revoke(l.id)}
                  >
                    Revogar
                  </button>
                )}
              </li>
            ))}
          </ul>
        )}
      </div>
    </>
  );
}
