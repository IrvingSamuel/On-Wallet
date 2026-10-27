import { FormEvent, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import {
  ApiError,
  api,
  type Collection,
  type CollectionDetail,
  type ListItem,
} from "../api/client";
import { EmptyState, formatDate } from "../components/ui";

export default function Collections() {
  const [items, setItems] = useState<Collection[]>([]);
  const [docs, setDocs] = useState<ListItem[]>([]);
  const [selectedId, setSelectedId] = useState<string>("");
  const [detail, setDetail] = useState<CollectionDetail | null>(null);
  const [name, setName] = useState("");
  const [rename, setRename] = useState("");
  const [addDocId, setAddDocId] = useState("");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function refreshList(preferId?: string) {
    const [colls, documents] = await Promise.all([
      api.listCollections(),
      api.listDocuments(),
    ]);
    setItems(colls.items);
    setDocs(documents.items);
    const nextId =
      preferId && colls.items.some((c) => c.id === preferId)
        ? preferId
        : colls.items[0]?.id ?? "";
    setSelectedId(nextId);
    if (!nextId) {
      setDetail(null);
      setRename("");
      return;
    }
    const d = await api.getCollection(nextId);
    setDetail(d);
    setRename(d.name);
  }

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");
    refreshList()
      .catch((e) => {
        if (!cancelled) {
          setError(e instanceof ApiError ? e.message : "Falha ao carregar coleções.");
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!selectedId) {
      setDetail(null);
      return;
    }
    let cancelled = false;
    setError("");
    api
      .getCollection(selectedId)
      .then((d) => {
        if (cancelled) return;
        setDetail(d);
        setRename(d.name);
      })
      .catch((e) => {
        if (cancelled) return;
        setDetail(null);
        setError(e instanceof ApiError ? e.message : "Falha ao abrir coleção.");
      });
    return () => {
      cancelled = true;
    };
  }, [selectedId]);

  const memberSet = useMemo(
    () => new Set(detail?.document_ids ?? []),
    [detail],
  );

  const availableDocs = useMemo(
    () => docs.filter((d) => !memberSet.has(d.id)),
    [docs, memberSet],
  );

  const members = useMemo(
    () => docs.filter((d) => memberSet.has(d.id)),
    [docs, memberSet],
  );

  async function create(e: FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    setBusy(true);
    setError("");
    try {
      const c = await api.createCollection(name.trim());
      setName("");
      await refreshList(c.id);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Falha ao criar coleção.");
    } finally {
      setBusy(false);
    }
  }

  async function saveRename(e: FormEvent) {
    e.preventDefault();
    if (!selectedId || !rename.trim()) return;
    setBusy(true);
    setError("");
    try {
      await api.renameCollection(selectedId, rename.trim());
      await refreshList(selectedId);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Falha ao renomear.");
    } finally {
      setBusy(false);
    }
  }

  async function removeCollection() {
    if (!selectedId) return;
    if (!window.confirm("Excluir esta coleção? Os documentos não serão apagados.")) return;
    setBusy(true);
    setError("");
    try {
      await api.deleteCollection(selectedId);
      await refreshList();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Falha ao excluir coleção.");
    } finally {
      setBusy(false);
    }
  }

  async function addDocument(e: FormEvent) {
    e.preventDefault();
    if (!selectedId || !addDocId) return;
    setBusy(true);
    setError("");
    try {
      const d = await api.addToCollection(selectedId, [addDocId]);
      setDetail(d);
      setAddDocId("");
      const colls = await api.listCollections();
      setItems(colls.items);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Falha ao adicionar documento.");
    } finally {
      setBusy(false);
    }
  }

  async function removeDocument(docId: string) {
    if (!selectedId) return;
    setBusy(true);
    setError("");
    try {
      await api.removeFromCollection(selectedId, docId);
      const d = await api.getCollection(selectedId);
      setDetail(d);
      const colls = await api.listCollections();
      setItems(colls.items);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Falha ao remover documento.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <div className="page-head">
        <h1>Coleções</h1>
      </div>
      <p className="muted">Agrupe documentos em coleções nomeadas. Um documento pode pertencer a várias.</p>

      {error && <div className="alert revoked">{error}</div>}

      <form className="card narrow" onSubmit={create} style={{ marginBottom: "1rem" }}>
        <label>
          Nova coleção
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Ex.: Pessoal, Trabalho"
            maxLength={120}
            required
          />
        </label>
        <button type="submit" className="btn primary" disabled={busy || !name.trim()}>
          Criar
        </button>
      </form>

      {loading ? (
        <p className="muted">Carregando coleções…</p>
      ) : items.length === 0 ? (
        <EmptyState title="Nenhuma coleção ainda">
          <p className="muted">Crie uma coleção acima e adicione documentos da carteira.</p>
        </EmptyState>
      ) : (
        <div style={{ display: "grid", gap: "1rem", gridTemplateColumns: "minmax(220px, 280px) 1fr" }}>
          <div className="card" style={{ padding: 0, overflow: "hidden" }}>
            <ul style={{ listStyle: "none", margin: 0, padding: 0 }}>
              {items.map((c) => (
                <li key={c.id}>
                  <button
                    type="button"
                    className="btn"
                    style={{
                      width: "100%",
                      justifyContent: "space-between",
                      borderRadius: 0,
                      border: "none",
                      borderBottom: "1px solid var(--border, #e5e7eb)",
                      background: c.id === selectedId ? "var(--surface-2, #f3f4f6)" : "transparent",
                    }}
                    onClick={() => setSelectedId(c.id)}
                  >
                    <span>{c.name}</span>
                    <span className="muted">{c.document_count ?? 0}</span>
                  </button>
                </li>
              ))}
            </ul>
          </div>

          {detail && (
            <div className="card">
              <form onSubmit={saveRename} style={{ display: "flex", gap: "0.5rem", flexWrap: "wrap", marginBottom: "1rem" }}>
                <input
                  value={rename}
                  onChange={(e) => setRename(e.target.value)}
                  maxLength={120}
                  aria-label="Nome da coleção"
                  style={{ flex: "1 1 200px" }}
                />
                <button type="submit" className="btn" disabled={busy || rename.trim() === detail.name}>
                  Renomear
                </button>
                <button type="button" className="btn danger" disabled={busy} onClick={removeCollection}>
                  Excluir
                </button>
              </form>
              <p className="muted" style={{ marginTop: 0 }}>
                Criada em {formatDate(detail.created_at)} · {detail.document_count} documento(s)
              </p>

              <form onSubmit={addDocument} style={{ display: "flex", gap: "0.5rem", flexWrap: "wrap", marginBottom: "1rem" }}>
                <select
                  value={addDocId}
                  onChange={(e) => setAddDocId(e.target.value)}
                  aria-label="Adicionar documento"
                  style={{ flex: "1 1 220px" }}
                >
                  <option value="">Adicionar documento…</option>
                  {availableDocs.map((d) => (
                    <option key={d.id} value={d.id}>
                      {d.title}
                    </option>
                  ))}
                </select>
                <button type="submit" className="btn primary" disabled={busy || !addDocId}>
                  Adicionar
                </button>
              </form>

              {members.length === 0 ? (
                <p className="muted">Nenhum documento nesta coleção.</p>
              ) : (
                <ul style={{ listStyle: "none", margin: 0, padding: 0, display: "grid", gap: "0.5rem" }}>
                  {members.map((d) => (
                    <li
                      key={d.id}
                      style={{
                        display: "flex",
                        justifyContent: "space-between",
                        gap: "0.75rem",
                        alignItems: "center",
                      }}
                    >
                      <Link to={`/documentos/${d.id}`}>{d.title}</Link>
                      <button
                        type="button"
                        className="btn danger sm"
                        disabled={busy}
                        onClick={() => removeDocument(d.id)}
                      >
                        Remover
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}
        </div>
      )}
    </>
  );
}
