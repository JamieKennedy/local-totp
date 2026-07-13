export type Algorithm = "SHA1" | "SHA256" | "SHA512";
export type CredentialSource = "manual" | "uri" | "generate";

export interface Status {
  setup: boolean;
  locked: boolean;
  authenticated: boolean;
  csrfToken?: string;
  version: string;
  testOnly: true;
}

export interface Credential {
  id: string;
  issuer: string;
  account: string;
  algorithm: Algorithm;
  digits: number;
  period: number;
  favorite: boolean;
  groupId?: string;
  tags: string[];
  notes: string;
  createdAt: string;
  updatedAt: string;
}

export interface CredentialInput {
  source: CredentialSource;
  uri?: string;
  issuer: string;
  account: string;
  secret?: string;
  algorithm: Algorithm;
  digits: number;
  period: number;
  favorite: boolean;
  groupId?: string;
  tags: string[];
  notes: string;
}

export interface CurrentCode {
  credentialId: string;
  code: string;
  validFrom: string;
  validUntil: string;
}

export interface CodesResponse {
  serverTime: string;
  codes: CurrentCode[];
}

export interface Group {
  id: string;
  name: string;
  color: string;
  createdAt: string;
  updatedAt: string;
}

export interface APIKey {
  id: string;
  name: string;
  createdAt: string;
  lastUsed?: string;
}

export interface CreatedAPIKey extends APIKey {
  key: string;
}

export interface SecretView {
  secret: string;
  uri: string;
}

export interface BackupPreview {
  id: string;
  credentials: number;
  groups: number;
  duplicates: number;
  nameConflicts: number;
  createdAt: string;
}
