import type { DocStatus } from "./api/client";

export const DOC_TYPES: Record<string, string> = {
  rg: "RG",
  cpf: "CPF",
  cnh: "CNH",
  certidao: "Certidão",
  certificado: "Certificado",
  diploma: "Diploma",
  comprovante: "Comprovante",
  outro: "Outro",
};

export const STATUS_LABEL: Record<DocStatus, string> = {
  active: "Ativo",
  replaced: "Substituído",
  revoked: "Revogado",
  expired: "Expirado",
};
