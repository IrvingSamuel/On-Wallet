import { useEffect, useState, type ReactNode } from "react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { api, type Health } from "../api/client";
import { useAuth } from "../auth";
import AssistantFab from "./AssistantFab";
import BrandLogo from "./BrandLogo";

const NAV = [
  { to: "/carteira", label: "Documentos" },
  { to: "/documentos/novo", label: "Novo documento" },
  { to: "/colecoes", label: "Coleções" },
  { to: "/verificar", label: "Verificar" },
  { to: "/auditoria", label: "Auditoria" },
];

function ApiStatus() {
  const [health, setHealth] = useState<Health | null>(null);
  const [down, setDown] = useState(false);
  useEffect(() => {
    api.health().then(setHealth).catch(() => setDown(true));
  }, []);
  const ok = health?.status === "ok";
  const label = down ? "API indisponível" : !health ? "verificando…" : ok ? "API ok" : "API degradada";
  return (
    <span
      className={`api-status ${ok ? "ok" : down || health ? "bad" : ""}`}
      title={health ? JSON.stringify(health.checks) : ""}
    >
      ● {label}
    </span>
  );
}

export default function Layout() {
  const [menuOpen, setMenuOpen] = useState(false);
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const displayName = user?.name?.trim() || user?.email?.split("@")[0] || "Titular";
  const initial = displayName.slice(0, 1).toUpperCase();

  useEffect(() => {
    document.body.style.overflow = menuOpen ? "hidden" : "";
    return () => {
      document.body.style.overflow = "";
    };
  }, [menuOpen]);

  async function onLogout() {
    await logout();
    navigate("/login", { replace: true });
  }

  return (
    <div className="app">
      <div className="flag-strip" aria-hidden="true" />

      <header className="topbar">
        <NavLink to="/carteira" className="brand" onClick={() => setMenuOpen(false)} aria-label="On-Wallet">
          <span className="brand-full"><BrandLogo size={32} /></span>
          <span className="brand-short"><BrandLogo size={28} short /></span>
        </NavLink>

        <nav className="topbar-nav desktop-nav" aria-label="Principal">
          {NAV.map((n) => (
            <NavLink key={n.to} to={n.to}>
              {n.label}
            </NavLink>
          ))}
        </nav>

        <div className="topbar-end desktop-nav">
          <ApiStatus />
          <div className="user-chip" title={user?.email ?? "Titular"}>
            <span className="avatar" aria-hidden="true">{initial}</span>
            <span className="user-name">{displayName}</span>
          </div>
          <button type="button" className="logout" onClick={onLogout}>Sair</button>
        </div>

        <button
          type="button"
          className="menu-toggle"
          aria-label={menuOpen ? "Fechar menu" : "Abrir menu"}
          aria-expanded={menuOpen}
          aria-controls="mobile-drawer"
          onClick={() => setMenuOpen((v) => !v)}
        >
          <span className="hamburger" data-open={menuOpen} />
        </button>
      </header>

      {menuOpen && (
        <button type="button" className="drawer-backdrop" aria-label="Fechar menu" onClick={() => setMenuOpen(false)} />
      )}
      <aside id="mobile-drawer" className={`drawer ${menuOpen ? "open" : ""}`} aria-hidden={!menuOpen}>
        <button
          type="button"
          className="drawer-close"
          aria-label="Fechar menu"
          onClick={() => setMenuOpen(false)}
        >
          <span aria-hidden="true">×</span>
        </button>
        <nav aria-label="Menu mobile">
          {NAV.map((n) => (
            <NavLink key={n.to} to={n.to} onClick={() => setMenuOpen(false)}>
              {n.label}
            </NavLink>
          ))}
        </nav>
        <div className="drawer-user">
          <ApiStatus />
          <div className="user-chip">
            <span className="avatar" aria-hidden="true">{initial}</span>
            <span>{displayName}</span>
          </div>
          <button
            type="button"
            className="btn ghost"
            onClick={() => {
              setMenuOpen(false);
              void onLogout();
            }}
          >
            Sair
          </button>
        </div>
      </aside>

      <main className="content">
        <Outlet />
      </main>

      <footer className="site-footer">
        <p>Projeto acadêmico – Project LAB, UNIMA Afya · Grupo 3</p>
        <p className="footer-links">
          <a href="#privacidade">Política de privacidade</a>
          <span aria-hidden="true">·</span>
          <a href="#termos">Termos de uso</a>
        </p>
        <p className="footer-note">
          Integridade (SHA-256) comprova que o arquivo não foi alterado; não comprova a autenticidade do emissor.
        </p>
      </footer>

      <AssistantFab />
    </div>
  );
}

/** Header azul + faixa para páginas públicas (login, /v/:token). */
export function PublicChrome({ children }: { children: ReactNode }) {
  return (
    <div className="app public-app">
      <div className="flag-strip" aria-hidden="true" />
      <header className="topbar public-topbar">
        <div className="brand" aria-label="On-Wallet">
          <span className="brand-full"><BrandLogo size={32} /></span>
          <span className="brand-short"><BrandLogo size={28} short /></span>
        </div>
      </header>
      <main className="content">{children}</main>
      <footer className="site-footer">
        <p>Projeto acadêmico – Project LAB, UNIMA Afya · Grupo 3</p>
        <p className="footer-note">
          Integridade (SHA-256) comprova que o arquivo não foi alterado; não comprova a autenticidade do emissor.
        </p>
      </footer>
    </div>
  );
}
