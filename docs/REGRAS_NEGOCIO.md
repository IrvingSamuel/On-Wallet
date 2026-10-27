# Regras de negócio — Project LAB (Carteira)

**Tipo:** as-built (regras **já enforced** no MVP).  
**Fontes:** [API.md](./API.md), [REQUISITOS.md](./REQUISITOS.md), [CASOS_DE_USO.md](./CASOS_DE_USO.md).  
**Origem acadêmica:** alinhadas às ~10 RNs do material *Carteira Digital* / quadro Chronos.

Cada regra: enunciado, justificativa, consequência se violada, rastreio RF/UC.

---

## RN-01 — Isolamento por titular autenticado

**Enunciado:** Operações privadas só valem para o titular da sessão (`owner_id` = UUID do usuário). Documento de outro titular é inacessível.

**Justificativa:** Privacidade e multi-tenant simples sem papéis de admin.

**Consequência:** Sem cookie → `401`; outro dono → `403 forbidden` (não vaza existência além do necessário).

**RF / UC:** RF-01, RF-02 · UC02, UC03, UC05–UC15

---

## RN-02 — Formato e tamanho do arquivo

**Enunciado:** Só entram PDF, JPEG ou PNG, com tamanho ≤ `MAX_UPLOAD_MB`. O MIME é detectado pelos magic bytes (extensão ignorada).

**Justificativa:** Reduz superfície de abuso e garante conteúdo processável pela carteira.

**Consequência:** MIME inválido → `415`; arquivo grande → `413`.

**RF / UC:** RF-03 · UC04, UC09

---

## RN-03 — Hash dos bytes exatos na inclusão

**Enunciado:** O SHA-256 é calculado sobre o conteúdo recebido e gravado em `integrity_records` na **mesma** transação do upload (junto com `documents` / `document_versions`). Se a transação falhar, o objeto MinIO é removido.

**Justificativa:** Prova de integridade amarra exatamente o que o titular enviou.

**Consequência:** Hash idêntico a `sha256sum` local; falha de DB não deixa objeto órfão útil.

**RF / UC:** RF-08, RF-09 · UC04

---

## RN-04 — Divergência não altera o registro

**Enunciado:** Na verificação, se o hash recalculado do MinIO divergir do registrado, o resultado é negativo (`divergent` / equivalente), mas o **hash armazenado não é sobrescrito**.

**Justificativa:** Preservar a evidência original; auditoria de verificação registra o evento.

**Consequência:** `POST …/verify` grava evidência; registro de integridade permanece.

**RF / UC:** RF-10, RF-11 · UC07

---

## RN-05 — Integridade ≠ autenticidade

**Enunciado:** Sucesso de verificação de hash **não** significa que o documento foi emitido por órgão oficial nem que a assinatura digital foi validada. A UI deixa isso explícito.

**Justificativa:** Escopo acadêmico do MVP é prova de não alteração do arquivo.

**Consequência:** Painéis / `AuthNote` e textos públicos (RF-23) separam os conceitos.

**RF / UC:** RF-23 · UC07, UC16

---

## RN-06 — Versões imutáveis

**Enunciado:** Substituir o arquivo cria uma **nova** versão (novo objeto MinIO + novo hash). Versões e objetos anteriores não são alterados no lugar.

**Justificativa:** Histórico auditável e verificação de versões antigas ainda possível no storage.

**Consequência:** `POST …/versions` só em estado elegível (ex.: `active`); revogado → `409`.

**RF / UC:** RF-13, RF-14 · UC09

---

## RN-07 — Revogação com motivo

**Enunciado:** Revogar documento exige motivo não vazio. Só estados que admitem revogação (`active` / `replaced`, conforme serviço) transitam para `revoked`.

**Justificativa:** Rastreabilidade do porquê da invalidação perante o titular.

**Consequência:** Sem motivo → `422`; estado inválido → `409`; documento continua consultável pelo titular.

**RF / UC:** RF-13, RF-15 · UC11

---

## RN-08 — Expiração automática sem mudar hash

**Enunciado:** Com `expires_at` no passado, o Sistema marca `status=expired` (job ~1 min e/ou lazy em list/get). O hash e o objeto MinIO **não** mudam.

**Justificativa:** Ciclo de vida temporal sem destruir a prova de integridade.

**Consequência:** Documento aparece como expirado; verificação de hash permanece possível.

**RF / UC:** RF-16 · UC17, UC05

---

## RN-09 — Compartilhamento com prazo e revogação

**Enunciado:** Links públicos usam token opaco amarrado a uma versão, com validade (ex.: 1/7/30 dias). Token expirado ou revogado → `410 gone`. Exclusão lógica do documento revoga shares ativos.

**Justificativa:** Terceiro valida só o que o titular autorizou, pelo tempo autorizado.

**Consequência:** Público sem sessão; sem acesso à carteira completa.

**RF / UC:** RF-18, RF-19, RF-20, RF-22 · UC14, UC16, UC12

---

## RN-10 — Exclusão lógica

**Enunciado:** `DELETE` preenche `deleted_at`; o documento some da listagem/detalhe do titular. Bytes no MinIO e hashes **permanecem**. Shares ativos são revogados.

**Justificativa:** Retirar da carteira sem apagar evidência forense de integridade.

**Consequência:** `204` no delete; GET posterior → `404`; audit `exclusao`.

**RF / UC:** RF-22 · UC12

---

## Rastreabilidade

| RN | RF principais | UC |
|---|---|---|
| RN-01 | 01, 02 | UC02–UC15 |
| RN-02 | 03 | UC04, UC09 |
| RN-03 | 08, 09 | UC04 |
| RN-04 | 10, 11 | UC07 |
| RN-05 | 23 | UC07, UC16 |
| RN-06 | 13, 14 | UC09 |
| RN-07 | 13, 15 | UC11 |
| RN-08 | 16 | UC17 |
| RN-09 | 18–20, 22 | UC12, UC14, UC16 |
| RN-10 | 22 | UC12 |
