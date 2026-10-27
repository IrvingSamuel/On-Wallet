import { useEffect, useState } from "react";
import { ApiError, api, type AuditEvent } from "../api/client";
import { formatDate } from "../components/ui";

const ACTION_LABEL: Record<string, string> = {
  inclusao: "Inclusão",
  substituicao: "Substituição",
  revogacao: "Revogação",
  correcao_metadados: "Correção de metadados",
  exclusao: "Exclusão",
  expiracao: "Expiração",
  link_gerado: "Link gerado",
  link_revogado: "Link revogado",
  verificacao: "Verificação",
};

export default function Audit() {
  const [items, setItems] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    api
      .listAudit()
      .then((r) => {
        if (!cancelled) setItems(r.items);
      })
      .catch((e) => {
        if (!cancelled) {
          setItems([]);
          setError(e instanceof ApiError ? e.message : "Falha ao carregar a auditoria.");
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <>
      <div className="page-head">
        <h1>Trilha de auditoria</h1>
      </div>

      {error && <div className="alert revoked">{error}</div>}

      {loading ? (
        <p className="muted">Carregando…</p>
      ) : items.length === 0 ? (
        <p className="muted">Nenhuma atividade registrada ainda.</p>
      ) : (
        <>
          <div className="card" style={{ padding: 0, overflow: "hidden" }}>
            <table className="table audit-table">
              <thead>
                <tr>
                  <th>Data/hora</th>
                  <th>Responsável</th>
                  <th>Ação</th>
                  <th>Documento</th>
                  <th>Contexto</th>
                </tr>
              </thead>
              <tbody>
                {items.map((a) => (
                  <tr key={a.id}>
                    <td>{formatDate(a.created_at)}</td>
                    <td>{a.actor}</td>
                    <td><span className="tag">{ACTION_LABEL[a.action] ?? a.action}</span></td>
                    <td>{a.document_title || "—"}</td>
                    <td className="muted">{a.context || "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="audit-cards" aria-label="Eventos de auditoria">
            {items.map((a) => (
              <article key={a.id} className="audit-card">
                <div className="when">{formatDate(a.created_at)} · {a.actor}</div>
                <span className="tag">{ACTION_LABEL[a.action] ?? a.action}</span>
                <div className="doc">{a.document_title || "—"}</div>
                <div className="ctx">{a.context || "—"}</div>
              </article>
            ))}
          </div>
        </>
      )}
    </>
  );
}
