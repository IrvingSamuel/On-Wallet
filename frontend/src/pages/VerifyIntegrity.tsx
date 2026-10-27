import { useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
  ApiError,
  api,
  type ListItem,
  type Verification,
  type VerifyResult,
} from "../api/client";
import { DocumentPreview } from "../components/DocumentPreview";
import {
  AuthNote,
  Hash,
  IntegrityResult,
  formatDateTime,
  type VerifyOutcome,
} from "../components/ui";

function toOutcome(result: VerifyResult): VerifyOutcome {
  if (result === "ok" || result === "divergent") return result;
  return "unverifiable";
}

function resultReason(v: Verification): string | undefined {
  if (v.result === "missing_record") return "Registro de integridade inexistente.";
  if (v.result === "unverifiable") return "Não foi possível ler o arquivo armazenado.";
  return undefined;
}

export default function VerifyIntegrity() {
  const [params, setParams] = useSearchParams();
  const docFromQuery = params.get("doc") ?? "";

  const [docs, setDocs] = useState<ListItem[]>([]);
  const [docId, setDocId] = useState(docFromQuery);
  const [outcome, setOutcome] = useState<VerifyOutcome | null>(null);
  const [last, setLast] = useState<Verification | null>(null);
  const [history, setHistory] = useState<Verification[]>([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const busyLock = useRef(false);

  useEffect(() => {
    api.listDocuments().then((r) => setDocs(r.items)).catch(() => setDocs([]));
  }, []);

  useEffect(() => {
    setDocId(docFromQuery);
  }, [docFromQuery]);

  useEffect(() => {
    if (!docId) {
      setHistory([]);
      return;
    }
    let cancelled = false;
    api
      .listVerifications(docId)
      .then((r) => {
        if (!cancelled) setHistory(r.items);
      })
      .catch(() => {
        if (!cancelled) setHistory([]);
      });
    return () => {
      cancelled = true;
    };
  }, [docId]);

  async function runVerify() {
    if (!docId || busyLock.current) return;
    busyLock.current = true;
    setBusy(true);
    setError("");
    setOutcome("checking");
    setLast(null);
    try {
      const rec = await api.verifyDocument(docId);
      setLast(rec);
      setOutcome(toOutcome(rec.result));
      const hist = await api.listVerifications(docId);
      setHistory(hist.items);
      setParams({ doc: docId }, { replace: true });
    } catch (e) {
      setOutcome(null);
      setError(
        e instanceof ApiError && e.status === 404
          ? "Documento não encontrado para este titular."
          : e instanceof ApiError
            ? e.message
            : "Falha ao verificar.",
      );
    } finally {
      busyLock.current = false;
      setBusy(false);
    }
  }

  const selected = docs.find((d) => d.id === docId);

  return (
    <>
      <div className="page-head">
        <h1>Verificar integridade</h1>
      </div>
      <p className="muted">
        Recalcula o SHA-256 do arquivo armazenado e compara com o registro. O hash registrado nunca é
        alterado em caso de divergência.
      </p>

      <div className="card narrow">
        <label>
          Documento
          <select
            value={docId}
            onChange={(e) => {
              setDocId(e.target.value);
              setOutcome(null);
              setLast(null);
              setError("");
              if (e.target.value) setParams({ doc: e.target.value }, { replace: true });
              else setParams({}, { replace: true });
            }}
            aria-label="Selecionar documento"
          >
            <option value="">Selecione…</option>
            {docs.map((d) => (
              <option key={d.id} value={d.id}>
                {d.title}
              </option>
            ))}
          </select>
        </label>

        {selected && (
          <>
            <p className="muted">
              <Link to={`/documentos/${selected.id}`}>{selected.title}</Link>
              {" · "}v{selected.version}
            </p>
            <DocumentPreview
              src={api.downloadUrl(selected.id)}
              mimeType={selected.mime_type}
              title={selected.title}
            />
          </>
        )}

        <button
          type="button"
          className="btn primary"
          disabled={!docId || busy}
          title={!docId ? "Selecione um documento" : undefined}
          onClick={runVerify}
        >
          {busy ? "Verificando…" : "Verificar agora"}
        </button>

        {error && <p className="error-text">{error}</p>}

        {outcome && (
          <IntegrityResult
            outcome={outcome}
            registeredAt={last?.created_at}
            reason={last ? resultReason(last) : undefined}
          />
        )}

        {last && last.expected_hash && (
          <dl className="kv" style={{ marginTop: "1rem" }}>
            <dt>Esperado</dt>
            <dd><Hash value={last.expected_hash} /></dd>
            {last.computed_hash && (
              <>
                <dt>Calculado</dt>
                <dd><Hash value={last.computed_hash} /></dd>
              </>
            )}
            <dt>Resultado</dt>
            <dd>{last.result}</dd>
          </dl>
        )}

        {outcome && outcome !== "checking" && <AuthNote />}
      </div>

      {history.length > 0 && (
        <div className="card">
          <h3>Histórico</h3>
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Data</th>
                  <th>Resultado</th>
                  <th>Versão</th>
                </tr>
              </thead>
              <tbody>
                {history.slice(0, 10).map((v) => (
                  <tr key={v.id}>
                    <td>{formatDateTime(v.created_at)}</td>
                    <td>{v.result}</td>
                    <td>v{v.version}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </>
  );
}
