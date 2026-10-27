# Política de Privacidade — On-Wallet

**Versão:** 1.0  
**Vigência:** 26 de outubro de 2026  
**Controlador (pessoa física):** Irving Samuel Lima Bandeira  
**Contato do titular de dados / encarregado (Art. 41 LGPD):** sync.irvingsamuel@gmail.com  
**Serviço:** On-Wallet / Carteira digital de documentos — https://carteira.eflow.space  
**Contexto:** projeto acadêmico Project LAB, UNIMA Afya · Grupo 3

Esta Política descreve como dados pessoais são tratados no On-Wallet, em conformidade com a Lei nº 13.709/2018 (LGPD) e, no que couber, com o Marco Civil da Internet (Lei nº. 12.965/2014).

---

## 1. Quem é o controlador

O controlador dos dados pessoais tratados neste serviço é a **pessoa física Irving Samuel Lima Bandeira**, responsável pelo ambiente de produção acadêmico em `carteira.eflow.space`.

Não há pessoa jurídica (CNPJ) associada a este serviço nesta versão. Pedidos de direitos do titular, dúvidas e reclamações devem ser enviados para **sync.irvingsamuel@gmail.com**.

## 2. Escopo

Aplica-se a:

- titulares com conta (cadastro/login);
- terceiros que acessam apenas links públicos de verificação (`/v/…`) ou recibo de integridade (`/i/…`);
- visitantes das páginas públicas de Política de Privacidade e Termos de Uso.

## 3. Dados pessoais tratados

| Categoria | Exemplos | Origem |
|---|---|---|
| Cadastro | nome, e-mail, hash de senha | informado pelo titular |
| Sessão | cookie HttpOnly `carteira_session` | gerado pelo sistema |
| Documentos | arquivos PDF/JPEG/PNG, metadados (título, tipo, tags, validade), hash SHA-256 | enviados pelo titular |
| Compartilhamento | tokens de link, prazos, status de revogação | ações do titular |
| Auditoria | eventos (upload, verify, revoke, delete etc.) com carimbo de tempo | gerados pelo sistema |
| Assistente (opcional) | trechos OCR do documento, mensagens de chat, respostas do modelo | uso voluntário do assistente |
| Técnico | logs de servidor, IP em registros de acesso quando aplicável, status de health | operação do serviço |

**Não coletamos** dados sensíveis de forma deliberada (Art. 5º, II, LGPD). Se o titular enviar documentos que contenham dados sensíveis (ex.: saúde, biométricos em CNH), o tratamento ocorre sob responsabilidade e base legal do próprio titular ao carregá-los no serviço acadêmico.

## 4. Finalidades e bases legais (Art. 7º LGPD)

| Finalidade | Base legal |
|---|---|
| Criar e autenticar conta; manter sessão | execução de procedimento preliminar / contrato de uso do serviço (Art. 7º, V) e consentimento no cadastro (Art. 7º, I) |
| Armazenar documentos, calcular e verificar integridade (SHA-256), gerar recibo, versões, revogação, coleções e auditoria | execução do serviço solicitado (Art. 7º, V) |
| Compartilhar verificação com terceiro via link/QR de prazo limitado | execução do serviço e legítimo interesse do titular em comprovar integridade (Art. 7º, V e IX), sem ampliar o acesso além do token |
| Assistente OCR + modelo de linguagem (coerência assistida) | consentimento pelo uso voluntário da função (Art. 7º, I) |
| Segurança, prevenção a abuso, continuidade do serviço | legítimo interesse (Art. 7º, IX), com salvaguardas |
| Cumprir obrigação legal ou ordem de autoridade | Art. 7º, II |

**Importante (RF-23):** a verificação de **integridade** (hash) **não** comprova autenticidade do emissor do documento nem substitui registros oficiais.

## 5. Com quem os dados podem ser compartilhados

Dados não são vendidos. Podem ser processados por operadores/infra necessários ao serviço:

- hospedagem do servidor e banco PostgreSQL no ambiente do controlador;
- armazenamento de objetos (MinIO/S3-compatível) para os arquivos;
- provedor de modelo de linguagem **Groq** (quando o assistente for usado) — conteúdo necessário à resposta pode ser enviado à API do provedor;
- membros do Grupo 3 / instituição acadêmica apenas no contexto de avaliação do Project LAB, sob dever de confidencialidade acadêmica, quando indispensável.

Terceiro verificador vê **somente** o que o link público expõe (metadados/resultado de integridade autorizados), sem acesso à carteira completa.

## 6. Transferência internacional

O uso do assistente (Groq) pode implicar processamento fora do Brasil. Nesse caso, o titular é informado nesta Política e o uso da função é **opcional**. Demais dados de produção destinam-se ao ambiente operado pelo controlador no Brasil, salvo necessidade técnica transitória de CDN/DNS.

## 7. Cookies e armazenamento local

- Cookie de sessão `carteira_session` (HttpOnly): essencial à autenticação — não é cookie de marketing.
- Não utilizamos cookies de publicidade ou rastreamento de terceiros para anúncios neste MVP.

## 8. Retenção e exclusão

- Conta e documentos: mantidos enquanto a conta existir e o serviço acadêmico estiver ativo, ou até exclusão solicitada pelo titular.
- Exclusão lógica de documento (`deleted_at`): remove da carteira ativa; arquivos/hash podem permanecer no armazenamento por prazo técnico/acadêmico limitado, sem reexposição na UI.
- Links de compartilhamento: expiram no prazo escolhido (1/7/30 dias) ou quando revogados.
- Logs de auditoria: retidos pelo tempo necessário à finalidade de segurança e demonstração acadêmica, podendo ser anonimizados ou eliminados ao encerrar o projeto.
- Após pedido de eliminação da conta, o controlador eliminará ou anonimizará os dados pessoais, salvo retenção exigida por lei ou para exercício regular de direitos.

## 9. Segurança

Medidas aplicadas no MVP incluem: senha armazenada com hash; cookie de sessão HttpOnly; isolamento multi-titular (`owner_id`); validação de MIME; HTTPS no domínio de produção; controle de acesso a MinIO/Postgres. Nenhum sistema é 100% seguro; o titular deve usar senha forte e não reutilizar credenciais.

## 10. Direitos do titular (Arts. 18 e 9º LGPD)

O titular pode solicitar, pelo e-mail **sync.irvingsamuel@gmail.com**:

- confirmação da existência de tratamento;
- acesso aos dados;
- correção de dados incompletos, inexatos ou desatualizados;
- anonimização, bloqueio ou eliminação de dados desnecessários ou tratados em desconformidade;
- portabilidade, quando aplicável e tecnicamente viável;
- eliminação dos dados tratados com consentimento (salvo hipóteses legais);
- informação sobre entidades com as quais o controlador compartilhou dados;
- informação sobre a possibilidade de não consentir e consequências;
- revogação do consentimento.

Resposta em prazo razoável (meta: até 15 dias). Também é possível apresentar reclamação à Autoridade Nacional de Proteção de Dados (ANPD).

## 11. Crianças e adolescentes

O serviço não é direcionado a menores de 18 anos. Contas de menores não devem ser criadas sem representação legal adequada.

## 12. Alterações

Esta Política pode ser atualizada para refletir mudanças no serviço ou na legislação. A versão vigente e a data constam no topo desta página e em `/privacidade`.

## 13. Contato

**Irving Samuel Lima Bandeira**  
E-mail: sync.irvingsamuel@gmail.com  
Serviço: https://carteira.eflow.space
