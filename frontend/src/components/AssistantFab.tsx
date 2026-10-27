import { useEffect, useRef, useState } from "react";
import Markdown from "react-markdown";
import { useLocation, useParams } from "react-router-dom";
import { ApiError, api, type ChatMessage } from "../api/client";

type UiMsg = { role: "user" | "assistant"; content: string; streaming?: boolean };

/** Carteira On-Wallet + óculos WALL-E — Wall-et bot. */
function WallEtMark({ size = 28 }: { size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 48 48"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      className="wallet-eyes"
    >
      <rect x="14" y="4" width="10" height="14" rx="1.5" fill="#FFCC29" />
      <rect x="18" y="2" width="10" height="14" rx="1.5" fill="#46AD44" />
      <rect x="22" y="3" width="11" height="14" rx="1.5" fill="#EAF1FA" />
      <path d="M25 7.5h5M25 10.5h4" stroke="#2966B1" strokeWidth="1.2" strokeLinecap="round" />
      <rect x="6" y="16" width="36" height="24" rx="4" fill="#2966B1" />
      <path d="M30 16h8a4 4 0 0 1 4 4v12a4 4 0 0 1-4 4h-8V16Z" fill="#1F4F8A" />
      <circle cx="40" cy="28" r="2" fill="#fff" />
      <g transform="translate(0 2)">
        <path d="M20 27h8" stroke="#C5CCD6" strokeWidth="1.8" strokeLinecap="round" />
        <circle cx="24" cy="27" r="1.6" fill="#C5CCD6" />
        <g transform="rotate(-10 15 28)">
          <ellipse cx="15" cy="28" rx="7.5" ry="8.5" fill="#DDE3EA" stroke="#9AA3B0" strokeWidth="1" />
          <ellipse cx="15" cy="28" rx="5" ry="5.8" fill="#1A2230" />
          <ellipse cx="15" cy="28" rx="2.8" ry="3.4" fill="#3D1A4A" />
          <circle cx="13.2" cy="26" r="1.4" fill="#fff" opacity="0.55" />
        </g>
        <g transform="rotate(10 33 28)">
          <ellipse cx="33" cy="28" rx="7.5" ry="8.5" fill="#DDE3EA" stroke="#9AA3B0" strokeWidth="1" />
          <ellipse cx="33" cy="28" rx="5" ry="5.8" fill="#1A2230" />
          <ellipse cx="33" cy="28" rx="2.8" ry="3.4" fill="#3D1A4A" />
          <circle cx="31.2" cy="26" r="1.4" fill="#fff" opacity="0.55" />
        </g>
      </g>
    </svg>
  );
}

function documentIdFromPath(pathname: string, paramId?: string): string | undefined {
  if (paramId && pathname.includes("/documentos/")) {
    return paramId;
  }
  const m = pathname.match(/^\/documentos\/([0-9a-f-]{36})/i);
  return m?.[1];
}

function BubbleContent({ role, content, streaming }: UiMsg) {
  if (role === "user") {
    return <>{content}</>;
  }
  return (
    <div className={`assistant-md${streaming ? " is-streaming" : ""}`}>
      <Markdown>{content || (streaming ? "…" : "")}</Markdown>
      {streaming && <span className="assistant-caret" aria-hidden="true" />}
    </div>
  );
}

export default function AssistantFab() {
  const location = useLocation();
  const { id: paramId } = useParams();
  const docId = documentIdFromPath(location.pathname, paramId);

  const [open, setOpen] = useState(false);
  const [input, setInput] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [messages, setMessages] = useState<UiMsg[]>([
    {
      role: "assistant",
      content:
        "Olá! Sou o **Wall-et bot**. Posso ajudar com a carteira (integridade, recibo, compartilhamento) e com coerência assistida do documento em foco.\n\nNão verifico autenticidade do emissor.",
    },
  ]);
  const listRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (open && listRef.current) {
      listRef.current.scrollTop = listRef.current.scrollHeight;
    }
  }, [messages, open]);

  async function send() {
    const text = input.trim();
    if (!text || busy) return;
    setInput("");
    setError("");
    const history: UiMsg[] = [...messages, { role: "user", content: text }];
    setMessages([...history, { role: "assistant", content: "", streaming: true }]);
    setBusy(true);
    try {
      const payload: ChatMessage[] = history.map((m) => ({ role: m.role, content: m.content }));
      await api.assistantChatStream(
        { messages: payload, document_id: docId },
        (delta) => {
          setMessages((prev) => {
            const copy = [...prev];
            const last = copy[copy.length - 1];
            if (last?.role === "assistant") {
              copy[copy.length - 1] = {
                role: "assistant",
                content: last.content + delta,
                streaming: true,
              };
            }
            return copy;
          });
        },
      );
      setMessages((prev) => {
        const copy = [...prev];
        const last = copy[copy.length - 1];
        if (last?.role === "assistant") {
          copy[copy.length - 1] = { role: "assistant", content: last.content, streaming: false };
        }
        return copy;
      });
    } catch (e) {
      setMessages((prev) => {
        const copy = [...prev];
        const last = copy[copy.length - 1];
        if (last?.role === "assistant" && last.streaming && !last.content) {
          copy.pop();
        } else if (last?.role === "assistant" && last.streaming) {
          copy[copy.length - 1] = { role: "assistant", content: last.content, streaming: false };
        }
        return copy;
      });
      setError(
        e instanceof ApiError && e.status === 503
          ? "Assistente indisponível (GROQ_API_KEY)."
          : e instanceof ApiError
            ? e.message
            : "Falha ao conversar com o assistente.",
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="assistant-fab-root">
      {open && (
        <div className="assistant-panel" role="dialog" aria-label="Wall-et bot">
          <header className="assistant-panel-head">
            <div className="assistant-panel-brand">
              <WallEtMark size={36} />
              <div>
                <strong>Wall-et bot</strong>
                <p className="muted">
                  {docId ? "Contexto: documento atual" : "Assistente On-Wallet"}
                </p>
              </div>
            </div>
            <button type="button" className="btn ghost" onClick={() => setOpen(false)} aria-label="Fechar">
              ×
            </button>
          </header>
          <div className="assistant-messages" ref={listRef}>
            {messages.map((m, i) => (
              <div key={i} className={`assistant-bubble ${m.role}`}>
                <BubbleContent {...m} />
              </div>
            ))}
          </div>
          {error && <p className="error-text assistant-error">{error}</p>}
          <form
            className="assistant-compose"
            onSubmit={(e) => {
              e.preventDefault();
              void send();
            }}
          >
            <input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder="Pergunte sobre a carteira…"
              aria-label="Mensagem para o assistente"
              disabled={busy}
            />
            <button type="submit" className="btn primary" disabled={busy || !input.trim()}>
              {busy ? "…" : "Enviar"}
            </button>
          </form>
        </div>
      )}
      <button
        type="button"
        className={`assistant-fab${open ? " is-open" : ""}`}
        aria-label={open ? "Fechar Wall-et bot" : "Abrir Wall-et bot"}
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
      >
        {open ? (
          <span className="assistant-fab-close" aria-hidden="true">×</span>
        ) : (
          <>
            <WallEtMark size={28} />
            <span className="assistant-fab-label">Wall-et</span>
          </>
        )}
      </button>
    </div>
  );
}
