import { Link } from "react-router-dom";
import { PublicChrome } from "../components/Layout";

const CONTROLLER = {
  name: "Irving Samuel Lima Bandeira",
  email: "sync.irvingsamuel@gmail.com",
  service: "https://carteira.eflow.space",
};

function LegalMeta({ kind }: { kind: "privacidade" | "termos" }) {
  return (
    <p className="legal-meta muted">
      Versão 1.0 · Vigência 26/10/2026 · Controlador (pessoa física):{" "}
      <strong>{CONTROLLER.name}</strong> · Contato:{" "}
      <a href={`mailto:${CONTROLLER.email}`}>{CONTROLLER.email}</a>
      {kind === "privacidade" ? " · LGPD (Lei 13.709/2018)" : null}
    </p>
  );
}

export function PrivacyPolicy() {
  return (
    <PublicChrome>
      <article className="card legal-doc">
        <div className="page-head">
          <h1>Política de Privacidade</h1>
          <Link className="btn outline" to="/termos">
            Ver Termos de Uso
          </Link>
        </div>
        <LegalMeta kind="privacidade" />
        <p>
          Esta Política descreve como dados pessoais são tratados no On-Wallet
          (Project LAB, UNIMA Afya · Grupo 3), em conformidade com a LGPD.
        </p>

        <h2>1. Quem é o controlador</h2>
        <p>
          O controlador é a <strong>pessoa física {CONTROLLER.name}</strong>,
          responsável pelo ambiente acadêmico em {CONTROLLER.service}. Não há
          CNPJ associado a este serviço nesta versão. Pedidos de direitos do
          titular:{" "}
          <a href={`mailto:${CONTROLLER.email}`}>{CONTROLLER.email}</a>.
        </p>

        <h2>2. Escopo</h2>
        <p>
          Aplica-se a titulares com conta, a terceiros que usam links públicos
          de verificação (<code>/v/…</code> ou <code>/i/…</code>) e a visitantes
          destas páginas legais.
        </p>

        <h2>3. Dados tratados</h2>
        <ul>
          <li>
            <strong>Cadastro:</strong> nome, e-mail, hash de senha
          </li>
          <li>
            <strong>Sessão:</strong> cookie HttpOnly <code>carteira_session</code>
          </li>
          <li>
            <strong>Documentos:</strong> PDF/JPEG/PNG, metadados, hash SHA-256
          </li>
          <li>
            <strong>Compartilhamento e auditoria:</strong> tokens, prazos, eventos
          </li>
          <li>
            <strong>Assistente (opcional):</strong> OCR, mensagens e respostas do
            modelo
          </li>
          <li>
            <strong>Técnico:</strong> logs de operação / health
          </li>
        </ul>
        <p>
          Não coletamos dados sensíveis de forma deliberada. Se você enviar
          documentos com dados sensíveis, o tratamento ocorre sob sua
          responsabilidade ao carregá-los no serviço acadêmico.
        </p>

        <h2>4. Finalidades e bases legais (Art. 7º LGPD)</h2>
        <ul>
          <li>
            Conta e sessão — execução do serviço (Art. 7º, V) e consentimento no
            cadastro (Art. 7º, I)
          </li>
          <li>
            Armazenamento, integridade SHA-256, recibo, versões, coleções,
            auditoria — execução do serviço (Art. 7º, V)
          </li>
          <li>
            Links públicos de verificação — execução do serviço / legítimo
            interesse (Art. 7º, V e IX)
          </li>
          <li>
            Assistente OCR+IA — consentimento pelo uso voluntário (Art. 7º, I)
          </li>
          <li>
            Segurança e continuidade — legítimo interesse (Art. 7º, IX)
          </li>
        </ul>
        <p>
          A verificação de <strong>integridade</strong> não comprova
          autenticidade do emissor (RF-23).
        </p>

        <h2>5. Compartilhamento</h2>
        <p>
          Dados não são vendidos. Podem ser processados por infraestrutura
          (Postgres, MinIO) e, se você usar o assistente, pela API Groq. Terceiros
          com link público veem apenas o autorizado pelo token.
        </p>

        <h2>6. Transferência internacional</h2>
        <p>
          O assistente (Groq) pode processar conteúdo fora do Brasil; o uso é
          opcional. Demais dados destinam-se ao ambiente operado pelo controlador
          no Brasil, salvo necessidade técnica transitória.
        </p>

        <h2>7. Cookies</h2>
        <p>
          Utilizamos cookie de sessão essencial à autenticação. Não usamos
          cookies de publicidade neste MVP.
        </p>

        <h2>8. Retenção e exclusão</h2>
        <p>
          Conta e documentos enquanto o serviço acadêmico e a conta existirem, ou
          até pedido de exclusão. Soft-delete remove da carteira ativa. Shares
          expiram ou são revogados. Contate o e-mail acima para eliminação/anonimização
          da conta, salvo retenção legal.
        </p>

        <h2>9. Segurança</h2>
        <p>
          Hash de senha, cookie HttpOnly, isolamento por titular, validação de
          MIME e HTTPS em produção. Use senha forte.
        </p>

        <h2>10. Seus direitos (Art. 18 LGPD)</h2>
        <p>
          Confirmação, acesso, correção, anonimização/bloqueio/eliminação,
          portabilidade quando viável, informação sobre compartilhamentos e
          revogação de consentimento — via{" "}
          <a href={`mailto:${CONTROLLER.email}`}>{CONTROLLER.email}</a>. Meta de
          resposta: até 15 dias. Também cabe reclamação à ANPD.
        </p>

        <h2>11. Menores</h2>
        <p>
          Serviço não direcionado a menores de 18 anos sem representação legal
          adequada.
        </p>

        <h2>12. Contato</h2>
        <p>
          <strong>{CONTROLLER.name}</strong>
          <br />
          <a href={`mailto:${CONTROLLER.email}`}>{CONTROLLER.email}</a>
          <br />
          {CONTROLLER.service}
        </p>
      </article>
    </PublicChrome>
  );
}

export function TermsOfUse() {
  return (
    <PublicChrome>
      <article className="card legal-doc">
        <div className="page-head">
          <h1>Termos de Uso</h1>
          <Link className="btn outline" to="/privacidade">
            Ver Política de Privacidade
          </Link>
        </div>
        <LegalMeta kind="termos" />
        <p>
          Ao criar conta ou utilizar o On-Wallet, você concorda com estes Termos
          e com a{" "}
          <Link to="/privacidade">Política de Privacidade</Link>.
        </p>

        <h2>1. Objeto</h2>
        <p>
          Carteira digital acadêmica para guardar documentos, registrar e
          verificar integridade (SHA-256), gerar recibo/QR, gerir versões e
          compartilhar links de prazo limitado. Não é cartório, autoridade
          certificadora nem substituto de sistemas oficiais (ex.: Gov.br).
        </p>

        <h2>2. Aceitação</h2>
        <p>
          Se não concordar, não crie conta nem envie documentos. Links públicos
          de verificação observam as mesmas limitações de responsabilidade.
        </p>

        <h2>3. Conta</h2>
        <ul>
          <li>Dados verdadeiros e senha confidencial</li>
          <li>Conta pessoal e intransferível</li>
          <li>Proibido acessar dados alheios ou abusar do serviço</li>
        </ul>

        <h2>4. Seu conteúdo</h2>
        <p>
          Você é responsável pela licitude dos arquivos enviados. O envio concede
          ao prestador apenas a licença necessária para armazenar, hashear,
          exibir, verificar e compartilhar conforme suas ações na interface.
        </p>

        <h2>5. Integridade ≠ autenticidade</h2>
        <p>
          O hash comprova coincidência de bytes com o registro — não a origem
          oficial nem a veracidade do conteúdo. O assistente de coerência é
          apoio informativo, não prova de autenticidade.
        </p>

        <h2>6. Compartilhamento</h2>
        <p>
          Ao gerar link ou QR, você autoriza quem tiver o token a ver o que a
          rota pública expõe, até expirar ou revogar.
        </p>

        <h2>7. Disponibilidade</h2>
        <p>
          Serviço acadêmico/experimental (“como está”), sem SLA comercial. Pode
          haver manutenção ou encerramento ao final do Project LAB.
        </p>

        <h2>8. Limitação de responsabilidade</h2>
        <p>
          Na máxima extensão legal, {CONTROLLER.name} e o Grupo 3 não respondem
          por danos indiretos ou decisões baseadas apenas na verificação de
          integridade ou no assistente. Responsabilidade por conteúdo ilícito
          enviada pelo usuário é do próprio usuário. Nada exclui dolo ou culpa
          onde a lei a torne irrenunciável.
        </p>

        <h2>9. Propriedade intelectual</h2>
        <p>
          Código, layout e materiais do On-Wallet pertencem aos respectivos
          titulares do projeto acadêmico.
        </p>

        <h2>10. Privacidade</h2>
        <p>
          Tratamento conforme a Política de Privacidade e a LGPD. Contato:{" "}
          <a href={`mailto:${CONTROLLER.email}`}>{CONTROLLER.email}</a>.
        </p>

        <h2>11. Lei e foro</h2>
        <p>
          Lei brasileira. Foro da comarca de residência de {CONTROLLER.name},
          Estado de Alagoas, com ressalva do art. 101, I, do CDC quando
          aplicável.
        </p>

        <h2>12. Contato</h2>
        <p>
          <strong>{CONTROLLER.name}</strong>
          <br />
          <a href={`mailto:${CONTROLLER.email}`}>{CONTROLLER.email}</a>
          <br />
          {CONTROLLER.service}
        </p>
      </article>
    </PublicChrome>
  );
}
