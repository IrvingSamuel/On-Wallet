import { useState } from "react";
import { Link } from "react-router-dom";
import { ApiError, api, type UploadResult } from "../api/client";
import { Hash, formatDate, formatSize } from "../components/ui";
import { DOC_TYPES } from "../constants";

const ACCEPT = ".pdf,.jpg,.jpeg,.png,application/pdf,image/jpeg,image/png";

export default function Upload() {
  const [file, setFile] = useState<File | null>(null);
  const [busy, setBusy] = useState(false);
  const [progress, setProgress] = useState(0);
  const [error, setError] = useState("");
  const [result, setResult] = useState<UploadResult | null>(null);

  async function submit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!file) return;
    const form = new FormData(e.currentTarget);
    form.set("file", file);

    setBusy(true);
    setError("");
    setProgress(15);
    const tick = window.setInterval(() => {
      setProgress((p) => Math.min(p + 12, 88));
    }, 180);

    try {
      const res = await api.upload(form);
      setProgress(100);
      setResult(res);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Falha de rede ao enviar o arquivo.");
      setProgress(0);
    } finally {
      clearInterval(tick);
      setBusy(false);
    }
  }

  if (result) {
    return (
      <div className="card narrow">
        <h1>Documento incluído</h1>
        <p className="ok-text">Arquivo armazenado e hash registrado.</p>
        <dl className="kv">
          <dt>Documento</dt><dd>{result.document.title}</dd>
          <dt>ID</dt><dd><code>{result.document.id}</code></dd>
          <dt>Arquivo</dt>
          <dd>{result.version.file_name} ({result.version.mime_type}, {formatSize(result.version.size_bytes)})</dd>
          <dt>Algoritmo</dt><dd>{result.integrity.algorithm}</dd>
          <dt>Hash</dt><dd><Hash value={result.integrity.hash} /></dd>
          <dt>Registrado em</dt><dd>{formatDate(result.integrity.created_at)}</dd>
        </dl>
        <div className="actions">
          <Link to={`/documentos/${result.document.id}`} className="btn primary">Ver documento</Link>
          <button type="button" className="btn ghost" onClick={() => { setResult(null); setFile(null); setProgress(0); }}>
            Enviar outro
          </button>
        </div>
      </div>
    );
  }

  const canSubmit = !!file && !busy;

  return (
    <form className="card narrow" onSubmit={submit}>
      <h1>Novo documento</h1>
      <p className="muted">PDF, JPEG ou PNG. O hash SHA-256 é calculado sobre o arquivo exato no envio.</p>

      <label className={`dropzone ${file ? "has-file" : ""}`}>
        <input
          type="file"
          accept={ACCEPT}
          onChange={(e) => { setFile(e.target.files?.[0] ?? null); setProgress(0); }}
          hidden
        />
        <span className="dropzone-icon" aria-hidden="true">⬆</span>
        {file ? (
          <>
            <strong>{file.name}</strong>
            <span>{formatSize(file.size)}</span>
          </>
        ) : (
          <span>Arraste ou clique para enviar (PDF, JPEG ou PNG)</span>
        )}
      </label>

      {busy && (
        <div className="progress" role="progressbar" aria-valuenow={progress} aria-valuemin={0} aria-valuemax={100}>
          <span style={{ width: `${progress}%` }} />
        </div>
      )}

      <label>
        Título
        <input name="title" placeholder={file?.name ?? "Ex.: RG frente e verso"} />
      </label>
      <div className="row">
        <label>
          Tipo *
          <select name="type" required defaultValue="">
            <option value="" disabled>Selecione</option>
            {Object.entries(DOC_TYPES).map(([k, v]) => (
              <option key={k} value={k}>{v}</option>
            ))}
          </select>
        </label>
        <label>
          Origem *
          <input name="origin" required placeholder="Ex.: SSP-AL, DETRAN, UFAL" />
        </label>
      </div>
      <label>
        Tags
        <input name="tags" placeholder="separadas por vírgula" />
      </label>
      <label>
        Validade (opcional)
        <input name="expires_at" type="date" />
      </label>

      {error && <p className="error-text">{error}</p>}
      <button
        type="submit"
        className="btn primary"
        disabled={!canSubmit}
        title={!file ? "Selecione um arquivo para enviar" : busy ? "Envio em andamento" : undefined}
      >
        {busy ? "Enviando…" : "Enviar documento"}
      </button>
    </form>
  );
}
