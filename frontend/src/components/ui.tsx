import { useState, type ReactNode } from "react";
import type { DocStatus } from "../api/client";
import { STATUS_LABEL } from "../constants";

export function StatusBadge({ status }: { status: DocStatus }) {
  const icons: Record<DocStatus, string> = {
    active: "✓",
    replaced: "↻",
    revoked: "✕",
    expired: "!",
  };
  return (
    <span className={`badge badge-${status}`}>
      <span className="badge-icon" aria-hidden="true">{icons[status]}</span>
      {STATUS_LABEL[status]}
    </span>
  );
}

export function Hash({ value, short }: { value: string; short?: boolean }) {
  const [copied, setCopied] = useState(false);
  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* ignore */
    }
  }
  const label =
    short && value.length > 16 ? `${value.slice(0, 8)}…${value.slice(-8)}` : value;
  return (
    <div className="hash-row">
      <code className="hash">{label}</code>
      <button type="button" className="btn ghost sm" onClick={copy} aria-label="Copiar hash">
        {copied ? "Copiado" : "Copiar"}
      </button>
    </div>
  );
}

export function formatDate(iso: string) {
  return new Date(iso).toLocaleString("pt-BR", { dateStyle: "short", timeStyle: "short" });
}

/** Inclui segundos — útil para diferenciar verificações no mesmo minuto. */
export function formatDateTime(iso: string) {
  return new Date(iso).toLocaleString("pt-BR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

export function formatDateOnly(iso: string) {
  return new Date(iso).toLocaleDateString("pt-BR");
}

/** KB = 1024 bytes (mesmo valor e unidade em toda a UI). */
export function formatSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 ** 2) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / 1024 ** 2).toFixed(1)} MB`;
}

export type VerifyOutcome = "checking" | "ok" | "divergent" | "unverifiable";

export function IntegrityResult({
  outcome,
  registeredAt,
  reason,
}: {
  outcome: VerifyOutcome;
  registeredAt?: string;
  reason?: string;
}) {
  if (outcome === "checking") {
    return (
      <div className="result-banner checking" role="status" aria-live="polite">
        <div className="spinner" aria-hidden="true" />
        <div>
          <h2>Verificando…</h2>
          <p>Comparando o arquivo com o registro de integridade.</p>
        </div>
      </div>
    );
  }
  if (outcome === "ok") {
    return (
      <div className="result-banner ok" role="status">
        <span className="result-icon" aria-hidden="true">✓</span>
        <div>
          <h2>Íntegro</h2>
          <p>
            Integridade preservada – o arquivo é idêntico ao registrado
            {registeredAt ? ` em ${formatDateOnly(registeredAt)}` : ""}.
          </p>
        </div>
      </div>
    );
  }
  if (outcome === "divergent") {
    return (
      <div className="result-banner bad" role="status">
        <span className="result-icon" aria-hidden="true">✕</span>
        <div>
          <h2>Divergente</h2>
          <p>Arquivo diferente do registrado.</p>
        </div>
      </div>
    );
  }
  return (
    <div className="result-banner warn" role="status">
      <span className="result-icon" aria-hidden="true">!</span>
      <div>
        <h2>Não verificável</h2>
        <p>{reason ?? "Não foi possível concluir a verificação."}</p>
      </div>
    </div>
  );
}

/** RF-23: deixa explícito que integridade ≠ autenticidade do emissor. */
export function AuthNote() {
  return (
    <p className="auth-note" role="note">
      <strong>Integridade ≠ autenticidade.</strong> Um hash íntegro confirma que o arquivo não
      mudou desde o registro; não confirma que o emissor é legítimo.
    </p>
  );
}

export function EmptyState({
  title,
  children,
  action,
}: {
  title: string;
  children?: ReactNode;
  action?: ReactNode;
}) {
  return (
    <div className="empty-state card">
      <div className="empty-illu" aria-hidden="true">
        <svg width="72" height="72" viewBox="0 0 72 72" fill="none">
          <rect x="14" y="10" width="44" height="52" rx="4" stroke="currentColor" strokeWidth="2" />
          <path d="M24 28h24M24 38h18M24 48h12" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
        </svg>
      </div>
      <h2>{title}</h2>
      {children}
      {action}
    </div>
  );
}
