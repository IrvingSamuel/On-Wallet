export type DocStatus = "active" | "replaced" | "revoked" | "expired";

export interface Document {
  id: string;
  owner_id: string;
  title: string;
  type: string;
  origin: string;
  tags: string[];
  status: DocStatus;
  revoked_reason?: string;
  revoked_at?: string;
  expires_at?: string | null;
  created_at: string;
}

export interface Version {
  id: string;
  document_id: string;
  version: number;
  object_key: string;
  file_name: string;
  size_bytes: number;
  mime_type: string;
  created_at: string;
}

export interface IntegrityRecord {
  id: string;
  version_id: string;
  version: number;
  algorithm: string;
  hash: string;
  created_at: string;
}

export interface UploadResult {
  document: Document;
  version: Version;
  integrity: IntegrityRecord;
}

export interface ListItem extends Document {
  file_name: string;
  mime_type: string;
  size_bytes: number;
  version: number;
}

export interface DocumentDetail {
  document: Document;
  version: Version;
  integrity?: IntegrityRecord;
}

export interface VersionSummary extends Version {
  algorithm?: string;
  hash?: string;
  current: boolean;
}

export interface ListParams {
  q?: string;
  type?: string;
  status?: string;
  tag?: string;
  collection?: string;
}

export interface Collection {
  id: string;
  owner_id: string;
  name: string;
  created_at: string;
  updated_at: string;
  document_count?: number;
}

export interface CollectionDetail extends Collection {
  document_ids: string[];
  document_count: number;
}

export type VerifyResult = "ok" | "divergent" | "unverifiable" | "missing_record";

export interface Verification {
  id: string;
  document_id: string;
  version_id: string;
  version: number;
  result: VerifyResult;
  algorithm: string;
  expected_hash?: string;
  computed_hash?: string;
  context: string;
  created_at: string;
}

export interface AuditEvent {
  id: string;
  owner_id: string;
  actor: string;
  action: string;
  document_id?: string;
  document_title: string;
  context: string;
  created_at: string;
}

export interface Health {
  status: "ok" | "degraded";
  checks: Record<string, string>;
}

export interface AuthUser {
  id: string;
  email: string;
  name: string;
  created_at: string;
}

export class ApiError extends Error {
  constructor(public status: number, public code: string, message: string) {
    super(message);
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api/v1${path}`, {
    credentials: "include",
    ...init,
    headers: init?.headers,
  });
  if (res.status === 204) {
    return undefined as T;
  }
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new ApiError(res.status, body?.error?.code ?? "unknown", body?.error?.message ?? res.statusText);
  }
  return body as T;
}

function qs(params: ListParams): string {
  const sp = new URLSearchParams();
  if (params.q) sp.set("q", params.q);
  if (params.type) sp.set("type", params.type);
  if (params.status) sp.set("status", params.status);
  if (params.tag) sp.set("tag", params.tag);
  if (params.collection) sp.set("collection", params.collection);
  const s = sp.toString();
  return s ? `?${s}` : "";
}

export const api = {
  health: () => request<Health>("/health"),
  me: () => request<AuthUser>("/auth/me"),
  login: (email: string, password: string) =>
    request<AuthUser>("/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    }),
  register: (email: string, password: string, name?: string) =>
    request<AuthUser>("/auth/register", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password, name: name ?? "" }),
    }),
  logout: async () => {
    const res = await fetch("/api/v1/auth/logout", { method: "POST", credentials: "include" });
    if (res.status === 204) return;
    const body = await res.json().catch(() => ({}));
    throw new ApiError(
      res.status,
      body?.error?.code ?? "unknown",
      body?.error?.message ?? res.statusText,
    );
  },
  documentTypes: () => request<{ types: string[] }>("/document-types"),
  integrity: (id: string) => request<IntegrityRecord>(`/documents/${id}/integrity`),
  listDocuments: (params: ListParams = {}) =>
    request<{ items: ListItem[]; total: number }>(`/documents${qs(params)}`),
  getDocument: (id: string) => request<DocumentDetail>(`/documents/${id}`),
  upload: (form: FormData) => request<UploadResult>("/documents", { method: "POST", body: form }),
  downloadUrl: (id: string, opts?: { download?: boolean }) =>
    `/api/v1/documents/${id}/file${opts?.download ? "?download=1" : ""}`,
  publicShareFileUrl: (token: string, opts?: { download?: boolean }) =>
    `/api/v1/public/shares/${encodeURIComponent(token)}/file${opts?.download ? "?download=1" : ""}`,
  publicIntegrityFileUrl: (id: string, opts?: { download?: boolean }) =>
    `/api/v1/public/documents/${encodeURIComponent(id)}/file${opts?.download ? "?download=1" : ""}`,
  getPublicIntegrity: (id: string) =>
    request<PublicIntegrity>(`/public/documents/${id}/integrity`),
  receiptUrl: (id: string) => `/api/v1/documents/${id}/receipt`,
  verifyDocument: (id: string) =>
    request<Verification>(`/documents/${id}/verify`, { method: "POST" }),
  listVerifications: (id: string) =>
    request<{ items: Verification[]; total: number }>(`/documents/${id}/verifications`),
  revokeDocument: (id: string, reason: string) =>
    request<Document>(`/documents/${id}/revoke`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ reason }),
    }),
  patchDocument: (
    id: string,
    body: {
      title: string;
      type: string;
      origin: string;
      tags: string[];
      expires_at: string | null;
    },
  ) =>
    request<Document>(`/documents/${id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  deleteDocument: async (id: string) => {
    const res = await fetch(`/api/v1/documents/${id}`, { method: "DELETE", credentials: "include" });
    if (res.status === 204) return;
    const body = await res.json().catch(() => ({}));
    throw new ApiError(
      res.status,
      body?.error?.code ?? "unknown",
      body?.error?.message ?? res.statusText,
    );
  },
  addVersion: (id: string, form: FormData) =>
    request<UploadResult>(`/documents/${id}/versions`, { method: "POST", body: form }),
  listVersions: (id: string) =>
    request<{ items: VersionSummary[]; total: number }>(`/documents/${id}/versions`),
  listAudit: () => request<{ items: AuditEvent[]; total: number }>("/audit"),
  createShare: (docId: string, days: number) =>
    request<ShareLink>(`/documents/${docId}/shares`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ days }),
    }),
  listShares: (docId: string) =>
    request<{ items: ShareLink[]; total: number }>(`/documents/${docId}/shares`),
  revokeShare: (shareId: string) =>
    request<ShareLink>(`/shares/${shareId}/revoke`, { method: "POST" }),
  getPublicShare: (token: string) => request<PublicShare>(`/public/shares/${token}`),
  listCollections: () =>
    request<{ items: Collection[]; total: number }>("/collections"),
  getCollection: (id: string) => request<CollectionDetail>(`/collections/${id}`),
  createCollection: (name: string) =>
    request<Collection>("/collections", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name }),
    }),
  renameCollection: (id: string, name: string) =>
    request<Collection>(`/collections/${id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name }),
    }),
  deleteCollection: async (id: string) => {
    const res = await fetch(`/api/v1/collections/${id}`, { method: "DELETE", credentials: "include" });
    if (res.status === 204) return;
    const body = await res.json().catch(() => ({}));
    throw new ApiError(
      res.status,
      body?.error?.code ?? "unknown",
      body?.error?.message ?? res.statusText,
    );
  },
  addToCollection: (id: string, documentIds: string[]) =>
    request<CollectionDetail>(`/collections/${id}/documents`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ document_ids: documentIds }),
    }),
  removeFromCollection: async (id: string, docId: string) => {
    const res = await fetch(`/api/v1/collections/${id}/documents/${docId}`, {
      method: "DELETE",
      credentials: "include",
    });
    if (res.status === 204) return;
    const body = await res.json().catch(() => ({}));
    throw new ApiError(
      res.status,
      body?.error?.code ?? "unknown",
      body?.error?.message ?? res.statusText,
    );
  },
  getAssist: (id: string) => request<AssistStatus>(`/documents/${id}/assist`),
  analyzeAssist: (id: string) =>
    request<AssistStatus>(`/documents/${id}/assist/analyze`, { method: "POST" }),
  assistantChat: (body: { messages: ChatMessage[]; document_id?: string }) =>
    request<ChatResponse>("/assistant/chat", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  /** SSE: chama onDelta a cada token; resolve com metadados finais. */
  async assistantChatStream(
    body: { messages: ChatMessage[]; document_id?: string },
    onDelta: (text: string) => void,
  ): Promise<ChatResponse> {
    const res = await fetch("/api/v1/assistant/chat?stream=1", {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        Accept: "text/event-stream",
      },
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      const errBody = await res.json().catch(() => ({}));
      throw new ApiError(
        res.status,
        errBody?.error?.code ?? "unknown",
        errBody?.error?.message ?? res.statusText,
      );
    }
    if (!res.body) {
      throw new ApiError(500, "internal", "stream vazio");
    }
    const reader = res.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    let reply = "";
    let citations: string[] | undefined;
    let disclaimer = "";
    // Cap visual typing at 50 chars/s (20ms/char), independent of network burst.
    const msPerChar = 1000 / 50;
    let nextEmitAt = 0;
    const emitPaced = async (chunk: string) => {
      for (const ch of chunk) {
        const now = performance.now();
        const wait = nextEmitAt - now;
        if (wait > 0) {
          await new Promise((r) => setTimeout(r, wait));
        }
        nextEmitAt = performance.now() + msPerChar;
        reply += ch;
        onDelta(ch);
      }
    };
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const parts = buffer.split("\n");
      buffer = parts.pop() ?? "";
      for (const rawLine of parts) {
        const line = rawLine.trim();
        if (!line.startsWith("data:")) continue;
        const payload = line.slice(5).trim();
        if (!payload || payload === "[DONE]") continue;
        let ev: { delta?: string; done?: boolean; citations?: string[]; disclaimer?: string; error?: string };
        try {
          ev = JSON.parse(payload);
        } catch {
          continue;
        }
        if (ev.error) {
          throw new ApiError(500, "assistant_error", ev.error);
        }
        if (ev.delta) {
          await emitPaced(ev.delta);
        }
        if (ev.done) {
          citations = ev.citations;
          disclaimer = ev.disclaimer ?? "";
        }
      }
    }
    return { reply, citations, disclaimer };
  },
};

export interface ShareLink {
  id: string;
  owner_id: string;
  document_id: string;
  version_id: string;
  token: string;
  expires_at: string;
  revoked_at?: string | null;
  created_at: string;
  revoked: boolean;
}

export interface PublicShare {
  title: string;
  document_id: string;
  version: number;
  mime_type: string;
  file_name: string;
  algorithm: string;
  hash: string;
  expires_at: string;
  verification: VerifyResult;
}

export interface PublicIntegrity {
  title: string;
  document_id: string;
  version: number;
  mime_type: string;
  file_name: string;
  algorithm: string;
  hash: string;
  registered_at: string;
  status: string;
  verification: VerifyResult;
}

export type ConsistencyResult = "consistent" | "divergent" | "unreadable" | "pending" | "error";
export type ExtractionStatus = "pending" | "running" | "done" | "unreadable" | "error";

export interface DocumentExtraction {
  id: string;
  document_id: string;
  version_id: string;
  owner_id: string;
  status: ExtractionStatus;
  raw_text?: string;
  fields_json: unknown;
  error?: string;
  created_at: string;
  updated_at: string;
}

export interface ConsistencyCheck {
  id: string;
  document_id: string;
  version_id: string;
  owner_id: string;
  result: ConsistencyResult;
  summary: string;
  details_json: unknown;
  created_at: string;
}

export interface AssistStatus {
  extraction?: DocumentExtraction;
  consistency?: ConsistencyCheck;
  disclaimer: string;
}

export interface ChatMessage {
  role: "user" | "assistant" | "system";
  content: string;
}

export interface ChatResponse {
  reply: string;
  citations?: string[];
  disclaimer: string;
}
