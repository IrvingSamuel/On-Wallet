import { FormEvent, useState } from "react";
import { Navigate, useLocation, useNavigate } from "react-router-dom";
import { ApiError, api } from "../api/client";
import { useAuth } from "../auth";
import BrandLogo from "../components/BrandLogo";

export default function Login() {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, loading, setUser } = useAuth();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const from = (location.state as { from?: string } | null)?.from || "/carteira";

  if (!loading && user) {
    return <Navigate to={from} replace />;
  }

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const u =
        mode === "login"
          ? await api.login(email.trim(), password)
          : await api.register(email.trim(), password, name.trim());
      setUser(u);
      navigate(from, { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Falha na autenticação.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="app">
      <div className="flag-strip" aria-hidden="true" />
      <div className="auth-split">
        <aside className="auth-hero">
          <BrandLogo stacked size={48} />
          <h1 className="visually-hidden">On-Wallet</h1>
          <p>Guarde, verifique e compartilhe documentos com prova de integridade.</p>
          <ul className="benefits">
            <li>Guardar PDF, JPEG e PNG com metadados e hash SHA-256</li>
            <li>Verificar se o arquivo continua idêntico ao registrado</li>
            <li>Compartilhar com link ou QR code de prazo limitado</li>
          </ul>
          <div className="benefits-mobile" aria-hidden="true">
            <span>Guardar</span>
            <span>·</span>
            <span>Verificar</span>
            <span>·</span>
            <span>Compartilhar</span>
          </div>
        </aside>

        <div className="auth-form-wrap">
          <form className="auth-card" onSubmit={submit}>
            <h2>{mode === "login" ? "Entrar" : "Criar conta"}</h2>
            {error && <div className="alert revoked">{error}</div>}
            {mode === "register" && (
              <label>
                Nome
                <input
                  name="name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  autoComplete="name"
                  placeholder="Como quer ser chamado"
                />
              </label>
            )}
            <label>
              E-mail
              <input
                type="email"
                name="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                autoComplete="username"
                required
                placeholder="seu@email.com"
              />
            </label>
            <label>
              Senha
              <input
                type="password"
                name="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete={mode === "login" ? "current-password" : "new-password"}
                required
                minLength={8}
                placeholder="Mínimo 8 caracteres"
              />
            </label>
            <button type="submit" className="btn primary" disabled={busy}>
              {busy ? "Aguarde…" : mode === "login" ? "Entrar" : "Cadastrar"}
            </button>
            <button
              type="button"
              className="btn outline"
              disabled={busy}
              onClick={() => {
                setMode(mode === "login" ? "register" : "login");
                setError("");
              }}
            >
              {mode === "login" ? "Criar conta" : "Já tenho conta"}
            </button>
            <p className="muted" style={{ marginTop: "0.75rem", fontSize: "0.85rem" }}>
              Links públicos de verificação (<code>/v/…</code>) não exigem login.
            </p>
          </form>
        </div>
      </div>
    </div>
  );
}
