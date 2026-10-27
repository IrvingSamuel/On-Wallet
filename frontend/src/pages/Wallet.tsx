import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { ApiError, api, type Collection, type DocStatus, type ListItem } from "../api/client";
import { DocumentPreview } from "../components/DocumentPreview";
import { EmptyState, StatusBadge, formatDate } from "../components/ui";
import { DOC_TYPES, STATUS_LABEL } from "../constants";

export default function Wallet() {
  const [q, setQ] = useState("");
  const [type, setType] = useState("");
  const [status, setStatus] = useState("");
  const [collection, setCollection] = useState("");
  const [collections, setCollections] = useState<Collection[]>([]);
  const [qDraft, setQDraft] = useState("");
  const [items, setItems] = useState<ListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    const t = window.setTimeout(() => setQ(qDraft.trim()), 250);
    return () => clearTimeout(t);
  }, [qDraft]);

  useEffect(() => {
    let cancelled = false;
    api
      .listCollections()
      .then((res) => {
        if (!cancelled) setCollections(res.items);
      })
      .catch(() => {
        if (!cancelled) setCollections([]);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");
    api
      .listDocuments({
        q: q || undefined,
        type: type || undefined,
        status: status || undefined,
        collection: collection || undefined,
      })
      .then((res) => {
        if (!cancelled) setItems(res.items);
      })
      .catch((e) => {
        if (!cancelled) {
          setItems([]);
          setError(e instanceof ApiError ? e.message : "Falha ao carregar a carteira.");
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [q, type, status, collection]);

  return (
    <>
      <div className="page-head">
        <h1>Meus documentos</h1>
        <Link to="/documentos/novo" className="btn primary">+ Adicionar documento</Link>
      </div>

      <div className="filters" role="search">
        <input
          placeholder="Buscar por título ou tag"
          value={qDraft}
          onChange={(e) => setQDraft(e.target.value)}
          aria-label="Buscar por título ou tag"
        />
        <select value={type} onChange={(e) => setType(e.target.value)} aria-label="Filtrar por tipo">
          <option value="">Todos os tipos</option>
          {Object.entries(DOC_TYPES).map(([k, v]) => (
            <option key={k} value={k}>{v}</option>
          ))}
        </select>
        <select value={status} onChange={(e) => setStatus(e.target.value)} aria-label="Filtrar por estado">
          <option value="">Todos os estados</option>
          {Object.entries(STATUS_LABEL).map(([k, v]) => (
            <option key={k} value={k}>{v}</option>
          ))}
        </select>
        <select
          value={collection}
          onChange={(e) => setCollection(e.target.value)}
          aria-label="Filtrar por coleção"
        >
          <option value="">Todas as coleções</option>
          {collections.map((c) => (
            <option key={c.id} value={c.id}>{c.name}</option>
          ))}
        </select>
      </div>

      {error && <div className="alert revoked">{error}</div>}

      {loading ? (
        <p className="muted">Carregando documentos…</p>
      ) : items.length === 0 ? (
        <EmptyState
          title="Você ainda não tem documentos"
          action={<Link to="/documentos/novo" className="btn primary">Adicionar documento</Link>}
        >
          <p className="muted">Envie um PDF, JPEG ou PNG para começar a carteira.</p>
        </EmptyState>
      ) : (
        <div className="doc-grid">
          {items.map((r) => (
            <Link key={r.id} to={`/documentos/${r.id}`} className="card doc-card">
              <DocumentPreview
                src={api.downloadUrl(r.id)}
                mimeType={r.mime_type}
                title={r.title}
                compact
              />
              <div className="doc-card-head">
                <span className="doc-type">
                  <span className="doc-type-icon" aria-hidden="true">
                    {(DOC_TYPES[r.type] ?? r.type).slice(0, 3)}
                  </span>
                  {DOC_TYPES[r.type] ?? r.type}
                </span>
                <StatusBadge status={r.status as DocStatus} />
              </div>
              <h3>{r.title}</h3>
              <div className="tags">
                {r.tags.map((t) => (
                  <span key={t} className="tag">#{t}</span>
                ))}
              </div>
              <small className="muted">Incluído em {formatDate(r.created_at)}</small>
            </Link>
          ))}
        </div>
      )}
    </>
  );
}
