import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { ApiError, api, type PublicShare, type VerifyResult } from "../api/client";
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

export default function Verify() {
  const { token = "" } = useParams();
  const [data, setData] = useState<PublicShare | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");
    setData(null);
    if (!token) {
      setError("Link inválido.");
      setLoading(false);
      return;
    }
    api
      .getPublicShare(token)
      .then((d) => {
        if (!cancelled) setData(d);
      })
      .catch((e) => {
        if (cancelled) return;
        if (e instanceof ApiError && (e.status === 410 || e.status === 404)) {
          setError("Este link é inválido, expirou ou foi revogado.");
        } else {
          setError(e instanceof ApiError ? e.message : "Falha ao validar o link.");
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [token]);

  return (
    <PublicChrome>
      <div className="public-verify">
        <h1>Validação de documento</h1>
        {loading && <p className="muted">Carregando…</p>}
        {!loading && error && (
          <div className="card">
            <IntegrityResult outcome="unverifiable" reason={error} />
            <p className="muted">
              Peça ao titular da carteira para gerar um novo link de compartilhamento.
              Links antigos não podem ser reativados após expiração ou revogação.
            </p>
          </div>
        )}
        {!loading && data && (
          <div className="card">
            <h2>{data.title}</h2>
            <p className="muted">
              Versão v{data.version} · válido até {formatDate(data.expires_at)}
              {data.file_name ? ` · ${data.file_name}` : ""}
            </p>
            <DocumentPreview
              src={api.publicShareFileUrl(token)}
              mimeType={data.mime_type || "application/octet-stream"}
              title={data.title}
            />
            <IntegrityResult
              outcome={toOutcome(data.verification)}
              reason={reasonFor(data.verification)}
            />
            <dl className="kv" style={{ marginTop: "1rem" }}>
              <dt>Algoritmo</dt>
              <dd>{data.algorithm || "—"}</dd>
              <dt>Hash</dt>
              <dd>{data.hash ? <Hash value={data.hash} /> : "—"}</dd>
            </dl>
            <p className="muted" style={{ fontSize: "0.875rem", marginTop: "1rem" }}>
              Este recibo comprova integridade do arquivo compartilhado; não comprova
              autenticidade do emissor.
            </p>
          </div>
        )}
      </div>
    </PublicChrome>
  );
}
