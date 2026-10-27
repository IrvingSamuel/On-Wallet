import { Navigate, Route, Routes } from "react-router-dom";
import { AuthProvider, RequireAuth } from "./auth";
import Layout from "./components/Layout";
import Login from "./pages/Login";
import Wallet from "./pages/Wallet";
import Upload from "./pages/Upload";
import DocumentDetail from "./pages/DocumentDetail";
import Share from "./pages/Share";
import Verify from "./pages/Verify";
import VerifyIntegrity from "./pages/VerifyIntegrity";
import PublicIntegrity from "./pages/PublicIntegrity";
import Receipt from "./pages/Receipt";
import Collections from "./pages/Collections";
import Audit from "./pages/Audit";
import { PrivacyPolicy, TermsOfUse } from "./pages/Legal";

export default function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/privacidade" element={<PrivacyPolicy />} />
        <Route path="/termos" element={<TermsOfUse />} />
        <Route path="/v/:token" element={<Verify />} />
        <Route path="/i/:id" element={<PublicIntegrity />} />
        <Route
          element={
            <RequireAuth>
              <Layout />
            </RequireAuth>
          }
        >
          <Route path="/carteira" element={<Wallet />} />
          <Route path="/documentos/novo" element={<Upload />} />
          <Route path="/documentos/:id" element={<DocumentDetail />} />
          <Route path="/documentos/:id/compartilhar" element={<Share />} />
          <Route path="/documentos/:id/recibo" element={<Receipt />} />
          <Route path="/verificar" element={<VerifyIntegrity />} />
          <Route path="/colecoes" element={<Collections />} />
          <Route path="/auditoria" element={<Audit />} />
        </Route>
        <Route path="*" element={<Navigate to="/carteira" replace />} />
      </Routes>
    </AuthProvider>
  );
}
