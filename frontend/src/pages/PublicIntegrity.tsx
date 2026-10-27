import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { ApiError, api, type PublicIntegrity as PublicIntegrityData, type VerifyResult } from "../api/client";
import { DocumentPreview } from "../components/DocumentPreview";
import { PublicChrome } from "../components/Layout";
import { Hash, IntegrityResult, formatDate, type VerifyOutcome } from "../components/ui";

function toOutcome(result: VerifyResult): VerifyOutcome {
  if (result === "ok" || result === "divergent") return result;
  return "unverifiable";
}

function reasonFor(result: VerifyResult): string | undefined {
  if (result === "missing_record") return "Registro de integridade inexistente.";
  if (result === "unverifiable") return "Não foi possível ler o arquivo armazenado.";
  return undefined;
}

export default function PublicIntegrity() {
  const { id = "" } = useParams();
  const [data, setData] = useState<PublicIntegrityData | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");
    setData(null);
    if (!id) {
      setError("Link inválido.");
      setLoading(false);
      return;
    }
    api
      .getPublicIntegrity(id)
      .then((d) => {
        if (!cancelled) setData(d);
      })
      .catch((e) => {
        if (cancelled) return;
        if (e instanceof ApiError && (e.status === 404 || e.status === 400)) {
          setError("Documento não encontrado ou indisponível.");
        } else {
          setError(e instanceof ApiError ? e.message : "Falha ao verificar o documento.");
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [id]);

  return (
    <PublicChrome>
      <div className="public-verify">
        <h1>Verificação de integridade</h1>
        {loading && <p className="muted">Verificando…</p>}
        {!loading && error && (
          <div className="card">
            <IntegrityResult outcome="unverifiable" reason={error} />
            <p className="muted">
              Confira se o recibo é o mais recente ou peça ao titular um novo link.
            </p>
          </div>
        )}
        {!loading && data && (
          <div className="card">
            <h2>{data.title}</h2>
            <p className="muted">
              Versão v{data.version}
              {data.file_name ? ` · ${data.file_name}` : ""}
              {data.registered_at ? ` · registrado em ${formatDate(data.registered_at)}` : ""}
            </p>
            <DocumentPreview
              src={api.publicIntegrityFileUrl(id)}
              mimeType={data.mime_type || "application/octet-stream"}
              title={data.title}
            />
            <IntegrityResult
              outcome={toOutcome(data.verification)}
              registeredAt={data.registered_at}
              reason={reasonFor(data.verification)}
            />
            <dl className="kv" style={{ marginTop: "1rem" }}>
              <dt>ID do documento</dt>
              <dd><code>{data.document_id}</code></dd>
              <dt>Algoritmo</dt>
              <dd>{data.algorithm || "—"}</dd>
              <dt>Hash</dt>
              <dd>{data.hash ? <Hash value={data.hash} /> : "—"}</dd>
            </dl>
            <p className="muted" style={{ fontSize: "0.875rem", marginTop: "1rem" }}>
              Este recibo comprova integridade do arquivo; não comprova autenticidade do emissor.
            </p>
          </div>
        )}
      </div>
    </PublicChrome>
  );
}
