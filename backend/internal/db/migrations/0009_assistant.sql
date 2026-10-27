-- Assistente: OCR + coerência assistida + RAG lexical (pós-MVP; RF-23 intacto).

CREATE TABLE document_extractions (
    id          uuid PRIMARY KEY,
    document_id uuid        NOT NULL REFERENCES documents(id),
    version_id  uuid        NOT NULL REFERENCES document_versions(id),
    owner_id    text        NOT NULL,
    status      text        NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending', 'running', 'done', 'unreadable', 'error')),
    raw_text    text        NOT NULL DEFAULT '',
    fields_json jsonb       NOT NULL DEFAULT '{}',
    error       text        NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (version_id)
);
CREATE INDEX document_extractions_doc_idx ON document_extractions (document_id, created_at DESC);

CREATE TABLE consistency_checks (
    id          uuid PRIMARY KEY,
    document_id uuid        NOT NULL REFERENCES documents(id),
    version_id  uuid        NOT NULL REFERENCES document_versions(id),
    owner_id    text        NOT NULL,
    result      text        NOT NULL
                CHECK (result IN ('consistent', 'divergent', 'unreadable', 'pending', 'error')),
    summary     text        NOT NULL DEFAULT '',
    details_json jsonb      NOT NULL DEFAULT '{}',
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX consistency_checks_doc_idx ON consistency_checks (document_id, created_at DESC);

CREATE TABLE assistant_chunks (
    id          uuid PRIMARY KEY,
    source      text        NOT NULL CHECK (source IN ('knowledge', 'document')),
    owner_id    text,
    document_id uuid REFERENCES documents(id),
    version_id  uuid REFERENCES document_versions(id),
    title       text        NOT NULL,
    body        text        NOT NULL,
    tsv         tsvector    NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX assistant_chunks_tsv_idx ON assistant_chunks USING GIN (tsv);
CREATE INDEX assistant_chunks_owner_idx ON assistant_chunks (owner_id) WHERE source = 'document';
CREATE INDEX assistant_chunks_knowledge_idx ON assistant_chunks (source) WHERE source = 'knowledge';

-- Seed knowledge (RAG lexical do produto).
INSERT INTO assistant_chunks (id, source, owner_id, document_id, version_id, title, body, tsv) VALUES
(
  'a0000001-0000-4000-8000-000000000001',
  'knowledge', NULL, NULL, NULL,
  'Integridade vs autenticidade',
  'A carteira prova integridade com SHA-256: o arquivo não mudou desde o registro. Isso NÃO prova autenticidade do órgão emissor nem que o documento é legítimo. Coerência assistida por IA só compara texto OCR com metadados declarados; nunca use o termo autenticidade verificada.',
  to_tsvector('portuguese', 'integridade autenticidade SHA-256 hash emissor coerência assistida OCR metadados RF-23')
),
(
  'a0000001-0000-4000-8000-000000000002',
  'knowledge', NULL, NULL, NULL,
  'Como verificar integridade',
  'Na tela do documento ou em Verificar, use Verificar agora. O sistema relê o objeto no MinIO, recalcula SHA-256 e compara com integrity_records. Divergência nunca altera o hash registrado. O recibo PDF tem QR para a tela pública /i/{id}.',
  to_tsvector('portuguese', 'verificar integridade recibo QR MinIO hash versão')
),
(
  'a0000001-0000-4000-8000-000000000003',
  'knowledge', NULL, NULL, NULL,
  'Compartilhamento público',
  'O titular gera um link com validade 1, 7 ou 30 dias. Terceiros abrem /v/{token} sem login e veem prévia + verificação de integridade. Links expirados ou revogados retornam erro. Excluir o documento revoga shares ativos.',
  to_tsvector('portuguese', 'compartilhar link token público validação terceiro revogar')
),
(
  'a0000001-0000-4000-8000-000000000004',
  'knowledge', NULL, NULL, NULL,
  'Upload e tipos de documento',
  'Aceita PDF, JPEG e PNG até o limite configurado. Tipos: rg, cpf, cnh, certidao, certificado, diploma, comprovante, outro. Metadados (título, origem, tags, validade) são informados pelo titular. Substituição cria nova versão imutável.',
  to_tsvector('portuguese', 'upload PDF JPEG PNG tipos CNH RG CPF diploma versão metadados')
),
(
  'a0000001-0000-4000-8000-000000000005',
  'knowledge', NULL, NULL, NULL,
  'Assistente e coerência assistida',
  'O assistente (FAB) responde dúvidas sobre a carteira e pode usar texto OCR do documento em contexto. Coerência assistida extrai campos do arquivo e compara com título/tipo/origem/tags. Resultados: consistent, divergent, unreadable, pending, error. Isso NÃO é prova de autenticidade do emissor.',
  to_tsvector('portuguese', 'assistente FAB chat OCR coerência consistent divergent unreadable RAG Groq')
),
(
  'a0000001-0000-4000-8000-000000000006',
  'knowledge', NULL, NULL, NULL,
  'Coleções e auditoria',
  'Coleções agrupam documentos do titular. A auditoria registra inclusão, substituição, revogação, verificação, correção de metadados e exclusão — sem conteúdo do arquivo.',
  to_tsvector('portuguese', 'coleções auditoria eventos revogação exclusão metadados')
);
