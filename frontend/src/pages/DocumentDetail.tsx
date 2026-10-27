import { useEffect, useRef, useState, type FormEvent } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import {
  ApiError,
  api,
  type AssistStatus,
  type ConsistencyResult,
  type DocStatus,
  type DocumentDetail as Detail,
  type IntegrityRecord,
  type Verification,
  type VerifyResult,
  type VersionSummary,
} from "../api/client";
import { DocumentPreview } from "../components/DocumentPreview";
import {
  Hash,
  IntegrityResult,
  StatusBadge,
  formatDate,
  formatDateTime,
  formatSize,
  type VerifyOutcome,
} from "../components/ui";
import { DOC_TYPES } from "../constants";

const ACCEPT = ".pdf,.jpg,.jpeg,.png,application/pdf,image/jpeg,image/png";

function toOutcome(result: VerifyResult): VerifyOutcome {
  if (result === "ok" || result === "divergent") return result;
  return "unverifiable";
}

function resultReason(v: Verification): string | undefined {
  if (v.result === "missing_record") return "Registro de integridade inexistente.";
  if (v.result === "unverifiable") return "Não foi possível ler o arquivo armazenado.";
  return undefined;
}

function consistencyLabel(r: ConsistencyResult): string {
  switch (r) {
    case "consistent":
      return "Coerente com metadados";
    case "divergent":
      return "Divergente dos metadados";
    case "unreadable":
      return "Texto insuficiente";
    case "pending":
      return "Pendente";
    case "error":
      return "Erro na análise";
    default:
      return r;
  }
}

function IntegrityPanel({
  integrity,
  lastVerify,
  verifying,
  onVerify,
  assist,
  analyzing,
  onAnalyze,
}: {
  integrity: IntegrityRecord | null;
  lastVerify: Verification | null;
  verifying: boolean;
  onVerify: () => void;
  assist: AssistStatus | null;
  analyzing: boolean;
  onAnalyze: () => void;
}) {
  const coherence = assist?.consistency;
  return (
    <div className="integrity">
      <div className="integrity-box ok">
        <div className="seal" aria-hidden="true">✓ Íntegro</div>
        <h4>Integridade</h4>
        {integrity ? (
          <>
            <p>Hash registrado ({integrity.algorithm}) para a versão {integrity.version}.</p>
            <Hash value={integrity.hash} />
            <small className="muted">Registrado em {formatDate(integrity.created_at)}</small>
          </>
        ) : (
          <p className="muted">Sem registro de integridade.</p>
        )}
        <button
          type="button"
          className="btn ghost"
          style={{ marginTop: "0.75rem" }}
          disabled={verifying}
          onClick={onVerify}
          title={verifying ? "Verificação em andamento" : undefined}
        >
          {verifying ? "Verificando…" : "Verificar agora"}
        </button>
        {verifying && <IntegrityResult outcome="checking" />}
        {!verifying && lastVerify && (
          <div style={{ marginTop: "0.75rem" }}>
            <IntegrityResult
              outcome={toOutcome(lastVerify.result)}
              registeredAt={integrity?.created_at}
              reason={resultReason(lastVerify)}
            />
            {lastVerify.computed_hash && (
              <small className="muted">
                Calculado: <code>{lastVerify.computed_hash.slice(0, 16)}…</code>
              </small>
            )}
          </div>
        )}
      </div>
      <div className={`integrity-box ${coherence?.result === "consistent" ? "ok" : "neutral"}`}>
        <h4>Coerência assistida</h4>
        {coherence ? (
          <>
            <p>
              <strong>{consistencyLabel(coherence.result)}.</strong> {coherence.summary}
            </p>
            <small className="muted">OCR/IA · {formatDateTime(coherence.created_at)}</small>
          </>
        ) : (
          <p className="muted">
            Ainda sem análise. Compara texto do arquivo com título, tipo e origem — não prova o emissor.
          </p>
        )}
        <button
          type="button"
          className="btn ghost"
          style={{ marginTop: "0.75rem" }}
          disabled={analyzing}
          onClick={onAnalyze}
        >
          {analyzing ? "Analisando…" : "Analisar com IA"}
        </button>
      </div>
      <div className="integrity-box neutral">
        <h4>Autenticidade</h4>
        <p>
          <strong>Não verificada.</strong> Integridade e coerência assistida não provam que o
          emissor é legítimo.
        </p>
      </div>
    </div>
  );
}

export default function DocumentDetail() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const fileRef = useRef<HTMLInputElement>(null);
  const [detail, setDetail] = useState<Detail | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [revokeOpen, setRevokeOpen] = useState(false);
  const [revokeReason, setRevokeReason] = useState("");
  const [revoking, setRevoking] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [history, setHistory] = useState<Verification[]>([]);
  const [versions, setVersions] = useState<VersionSummary[]>([]);
  const [lastVerify, setLastVerify] = useState<Verification | null>(null);
  const [verifying, setVerifying] = useState(false);
  const [assist, setAssist] = useState<AssistStatus | null>(null);
  const [analyzing, setAnalyzing] = useState(false);
  const verifyingLock = useRef(false);
  const [replacing, setReplacing] = useState(false);
  const [editing, setEditing] = useState(false);
  const [savingMeta, setSavingMeta] = useState(false);
  const [metaTitle, setMetaTitle] = useState("");
  const [metaType, setMetaType] = useState("");
  const [metaOrigin, setMetaOrigin] = useState("");
  const [metaTags, setMetaTags] = useState("");
  const [metaExpires, setMetaExpires] = useState("");

  function loadAll() {
    return Promise.all([
      api.getDocument(id),
      api.listVerifications(id),
      api.listVersions(id),
    ]);
  }

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");
    setLastVerify(null);
    setAssist(null);
    loadAll()
      .then(([d, hist, vers]) => {
        if (cancelled) return;
        setDetail(d);
        setMetaTitle(d.document.title);
        setMetaType(d.document.type);
        setMetaOrigin(d.document.origin);
        setMetaTags(d.document.tags.join(", "));
        setMetaExpires(d.document.expires_at ? d.document.expires_at.slice(0, 10) : "");
        setEditing(false);
        setHistory(hist.items);
        setVersions(vers.items);
        if (hist.items[0]) setLastVerify(hist.items[0]);
      })
      .catch((e) => {
        if (!cancelled) {
          setDetail(null);
          setHistory([]);
          setVersions([]);
          setError(
            e instanceof ApiError && (e.status === 404 || e.status === 400)
              ? "Documento não encontrado para este titular."
              : "Falha ao consultar a API.",
          );
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    api
      .getAssist(id)
      .then((st) => {
        if (!cancelled) setAssist(st);
      })
      .catch(() => {
        if (!cancelled) setAssist(null);
      });
    return () => {
      cancelled = true;
    };
  }, [id]);

  async function runAnalyze() {
    if (analyzing) return;
    setAnalyzing(true);
    setError("");
    try {
      const st = await api.analyzeAssist(id);
      setAssist(st);
    } catch (e) {
      setError(
        e instanceof ApiError && e.status === 503
          ? "Assistente indisponível (GROQ_API_KEY)."
          : e instanceof ApiError
            ? e.message
            : "Falha na análise assistida.",
      );
    } finally {
      setAnalyzing(false);
    }
  }

  async function runVerify() {
    if (verifyingLock.current) return;
    verifyingLock.current = true;
    setVerifying(true);
    setError("");
    try {
      const rec = await api.verifyDocument(id);
      setLastVerify(rec);
      const hist = await api.listVerifications(id);
      setHistory(hist.items);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Falha ao verificar.");
    } finally {
      verifyingLock.current = false;
      setVerifying(false);
    }
  }

  async function confirmRevoke() {
    if (!revokeReason.trim() || !detail) return;
    setRevoking(true);
    setError("");
    try {
      const doc = await api.revokeDocument(id, revokeReason.trim());
      setDetail({ ...detail, document: doc });
      setRevokeOpen(false);
      setRevokeReason("");
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Falha ao revogar.");
    } finally {
      setRevoking(false);
    }
  }

  async function onReplaceFile(file: File | null) {
    if (!file) return;
    setReplacing(true);
    setError("");
    try {
      const form = new FormData();
      form.set("file", file);
      await api.addVersion(id, form);
      const [d, hist, vers] = await loadAll();
      setDetail(d);
      setHistory(hist.items);
      setVersions(vers.items);
      setLastVerify(null);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Falha ao substituir.");
    } finally {
      setReplacing(false);
      if (fileRef.current) fileRef.current.value = "";
    }
  }

  async function confirmDelete() {
    if (!detail || deleting) return;
    setDeleting(true);
    setError("");
    try {
      await api.deleteDocument(id);
      navigate("/carteira");
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Falha ao excluir.");
      setDeleting(false);
    }
  }

  async function saveMetadata(e: FormEvent) {
    e.preventDefault();
    if (!detail || savingMeta) return;
    setSavingMeta(true);
    setError("");
    try {
      const tags = metaTags
        .split(",")
        .map((t) => t.trim())
        .filter(Boolean);
      const doc = await api.patchDocument(id, {
        title: metaTitle.trim() || detail.document.title,
        type: metaType,
        origin: metaOrigin.trim(),
        tags,
        expires_at: metaExpires.trim() || null,
      });
      setDetail({ ...detail, document: doc });
      setMetaTitle(doc.title);
      setMetaType(doc.type);
      setMetaOrigin(doc.origin);
      setMetaTags(doc.tags.join(", "));
      setMetaExpires(doc.expires_at ? doc.expires_at.slice(0, 10) : "");
      setEditing(false);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Falha ao salvar metadados.");
    } finally {
      setSavingMeta(false);
    }
  }

  const title = detail?.document.title ?? "Documento";
  const status: DocStatus = detail?.document.status ?? "active";
  const isRevoked = status === "revoked";
  const canRevoke = status === "active" || status === "replaced";
  const canReplace = status === "active";
  const canEditMeta = status === "active" || status === "replaced" || status === "expired";

  return (
    <>
      <div className="page-head">
        <div>
          <Link to="/carteira" className="muted">← Documentos</Link>
          <h1>
            {title} {detail && <StatusBadge status={status} />}
          </h1>
        </div>
        <Link to={`/documentos/${id}/compartilhar`} className="btn primary">Compartilhar</Link>
      </div>

      {error && <div className="alert revoked">{error}</div>}
      {isRevoked && detail?.document.revoked_reason && (
        <div className="alert revoked">
          Revogado
          {detail.document.revoked_at ? ` em ${formatDate(detail.document.revoked_at)}` : ""}
          : {detail.document.revoked_reason}
        </div>
      )}

      <input
        ref={fileRef}
        type="file"
        accept={ACCEPT}
        hidden
        onChange={(e) => onReplaceFile(e.target.files?.[0] ?? null)}
      />

      <div className="actions">
        {detail ? (
          <a className="btn ghost" href={api.downloadUrl(id, { download: true })} download>
            Baixar
          </a>
        ) : (
          <button type="button" className="btn ghost" disabled>Baixar</button>
        )}
        <button
          type="button"
          className="btn ghost"
          disabled={!detail || verifying}
          onClick={runVerify}
          title={verifying ? "Verificação em andamento" : "Recalcular hash do arquivo armazenado"}
        >
          {verifying ? "Verificando…" : "Verificar agora"}
        </button>
        {canReplace && (
          <button
            type="button"
            className="btn ghost"
            disabled={replacing}
            onClick={() => fileRef.current?.click()}
            title="Enviar nova versão (PDF, JPEG ou PNG)"
          >
            {replacing ? "Substituindo…" : "Substituir"}
          </button>
        )}
        {canRevoke && (
          <button
            type="button"
            className="btn danger"
            onClick={() => setRevokeOpen(true)}
            title="Revogar documento (exige motivo)"
          >
            Revogar
          </button>
        )}
        {detail && (
          <button
            type="button"
            className="btn danger"
            onClick={() => setDeleteOpen(true)}
            title="Excluir logicamente (some da carteira)"
          >
            Excluir
          </button>
        )}
        <Link to={`/documentos/${id}/recibo`} className="btn ghost">Recibo</Link>
        <Link to={`/verificar?doc=${id}`} className="btn ghost">Abrir verificação</Link>
      </div>

      {deleteOpen && detail && (
        <div className="card">
          <h3>Excluir documento</h3>
          <p className="muted">
            O documento deixa de aparecer na carteira (exclusão lógica). O hash e o arquivo
            permanecem armazenados para auditoria.
          </p>
          <div className="actions">
            <button
              type="button"
              className="btn danger"
              disabled={deleting}
              onClick={confirmDelete}
            >
              {deleting ? "Excluindo…" : "Confirmar exclusão"}
            </button>
            <button
              type="button"
              className="btn ghost"
              disabled={deleting}
              onClick={() => setDeleteOpen(false)}
            >
              Cancelar
            </button>
          </div>
        </div>
      )}

      {revokeOpen && canRevoke && (
        <div className="card">
          <h3>Revogar documento</h3>
          <p className="muted">Informe o motivo. O documento permanece consultável para auditoria.</p>
          <label>
            Motivo *
            <textarea
              rows={3}
              value={revokeReason}
              onChange={(e) => setRevokeReason(e.target.value)}
              required
              placeholder="Ex.: emitido com nome incorreto"
            />
          </label>
          <div className="actions">
            <button
              type="button"
              className="btn danger"
              disabled={!revokeReason.trim() || revoking}
              title={!revokeReason.trim() ? "Informe o motivo para revogar" : undefined}
              onClick={confirmRevoke}
            >
              {revoking ? "Revogando…" : "Confirmar revogação"}
            </button>
            <button type="button" className="btn ghost" disabled={revoking} onClick={() => setRevokeOpen(false)}>
              Cancelar
            </button>
          </div>
        </div>
      )}

      {loading ? (
        <p className="muted">Carregando…</p>
      ) : detail ? (
        <>
          <div className="card">
            <h3>Pré-visualização</h3>
            <DocumentPreview
              src={api.downloadUrl(id)}
              mimeType={detail.version.mime_type}
              title={detail.document.title}
            />
          </div>
          <div className="detail-grid">
            <div className="card">
              <div className="page-head" style={{ marginBottom: "0.75rem" }}>
                <h3 style={{ margin: 0 }}>Metadados</h3>
                {canEditMeta && !editing && (
                  <button type="button" className="btn ghost" onClick={() => setEditing(true)}>
                    Editar
                  </button>
                )}
              </div>
              {editing && canEditMeta ? (
                <form onSubmit={saveMetadata}>
                  <label>
                    Título
                    <input value={metaTitle} onChange={(e) => setMetaTitle(e.target.value)} required />
                  </label>
                  <label>
                    Tipo
                    <select value={metaType} onChange={(e) => setMetaType(e.target.value)} required>
                      {Object.entries(DOC_TYPES).map(([k, v]) => (
                        <option key={k} value={k}>{v}</option>
                      ))}
                    </select>
                  </label>
                  <label>
                    Origem
                    <input value={metaOrigin} onChange={(e) => setMetaOrigin(e.target.value)} required />
                  </label>
                  <label>
                    Tags
                    <input
                      value={metaTags}
                      onChange={(e) => setMetaTags(e.target.value)}
                      placeholder="separadas por vírgula"
                    />
                  </label>
                  <label>
                    Validade
                    <input
                      type="date"
                      value={metaExpires}
                      onChange={(e) => setMetaExpires(e.target.value)}
                    />
                  </label>
                  <div className="actions">
                    <button type="submit" className="btn primary" disabled={savingMeta}>
                      {savingMeta ? "Salvando…" : "Salvar"}
                    </button>
                    <button
                      type="button"
                      className="btn ghost"
                      disabled={savingMeta}
                      onClick={() => {
                        setEditing(false);
                        setMetaTitle(detail.document.title);
                        setMetaType(detail.document.type);
                        setMetaOrigin(detail.document.origin);
                        setMetaTags(detail.document.tags.join(", "));
                        setMetaExpires(detail.document.expires_at ? detail.document.expires_at.slice(0, 10) : "");
                      }}
                    >
                      Cancelar
                    </button>
                  </div>
                </form>
              ) : (
                <dl className="kv">
                  <dt>ID</dt><dd><code>{detail.document.id}</code></dd>
                  <dt>Tipo</dt><dd>{DOC_TYPES[detail.document.type] ?? detail.document.type}</dd>
                  <dt>Origem</dt><dd>{detail.document.origin}</dd>
                  <dt>Tags</dt><dd>{detail.document.tags.join(", ") || "-"}</dd>
                  <dt>Arquivo</dt>
                  <dd>
                    {detail.version.file_name} ({detail.version.mime_type}, {formatSize(detail.version.size_bytes)})
                  </dd>
                  <dt>Versão</dt><dd>v{detail.version.version}</dd>
                  <dt>Incluído em</dt><dd>{formatDate(detail.document.created_at)}</dd>
                  <dt>Validade</dt>
                  <dd>{detail.document.expires_at ? formatDate(detail.document.expires_at) : "Sem validade"}</dd>
                </dl>
              )}
            </div>
            <IntegrityPanel
              integrity={detail.integrity ?? null}
              lastVerify={lastVerify}
              verifying={verifying}
              onVerify={runVerify}
              assist={assist}
              analyzing={analyzing}
              onAnalyze={runAnalyze}
            />
          </div>
          {versions.length > 0 && (
            <div className="card">
              <h3>Versões</h3>
              <div className="table-wrap">
                <table className="table">
                  <thead>
                    <tr>
                      <th>Versão</th>
                      <th>Arquivo</th>
                      <th>Hash</th>
                      <th>Data</th>
                      <th>Estado</th>
                    </tr>
                  </thead>
                  <tbody>
                    {versions.map((v) => (
                      <tr key={v.id}>
                        <td>v{v.version}</td>
                        <td>{v.file_name} ({formatSize(v.size_bytes)})</td>
                        <td>{v.hash ? <Hash value={v.hash} /> : "—"}</td>
                        <td>{formatDate(v.created_at)}</td>
                        <td>{v.current ? "Atual" : "Substituída"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
          {history.length > 0 && (
            <div className="card">
              <h3>Histórico de verificações</h3>
              <div className="table-wrap">
                <table className="table">
                  <thead>
                    <tr>
                      <th>Data</th>
                      <th>Resultado</th>
                      <th>Versão</th>
                      <th>Hash calculado</th>
                    </tr>
                  </thead>
                  <tbody>
                    {history.slice(0, 10).map((v) => (
                      <tr key={v.id}>
                        <td>{formatDateTime(v.created_at)}</td>
                        <td>{v.result}</td>
                        <td>v{v.version}</td>
                        <td>{v.computed_hash ? <Hash value={v.computed_hash} short /> : "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </>
      ) : null}
    </>
  );
}
