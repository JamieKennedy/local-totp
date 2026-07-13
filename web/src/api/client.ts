import type {
  APIKey,
  BackupPreview,
  CodesResponse,
  CreatedAPIKey,
  Credential,
  CredentialInput,
  Group,
  SecretView,
  Status,
} from "./types";

let csrfToken = "";

export class APIError extends Error {
  readonly code: string;
  readonly status: number;

  constructor(code: string, message: string, status: number) {
    super(message);
    this.name = "APIError";
    this.code = code;
    this.status = status;
  }
}

export function setCSRF(value: string | undefined): void {
  csrfToken = value ?? "";
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body !== undefined && !(init.body instanceof FormData)) {
    headers.set("Content-Type", "application/json");
  }
  if (init.method !== undefined && init.method !== "GET" && csrfToken !== "") {
    headers.set("X-CSRF-Token", csrfToken);
  }
  const response = await fetch(`/api/v1${path}`, { ...init, headers, credentials: "same-origin" });
  if (!response.ok) {
    const failure = (await response.json().catch(() => ({
      error: { code: "request_failed", message: response.statusText },
    }))) as { error: { code: string; message: string } };
    throw new APIError(failure.error.code, failure.error.message, response.status);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return (await response.json()) as T;
}

export const api = {
  async status(): Promise<Status> {
    const value = await request<Status>("/status");
    setCSRF(value.csrfToken);
    return value;
  },
  async setup(password: string): Promise<{ recoveryKey: string; csrfToken: string }> {
    const value = await request<{ recoveryKey: string; csrfToken: string }>("/setup", {
      method: "POST",
      body: JSON.stringify({ password }),
    });
    setCSRF(value.csrfToken);
    return value;
  },
  async unlock(password: string): Promise<void> {
    const value = await request<{ csrfToken: string }>("/session/unlock", {
      method: "POST",
      body: JSON.stringify({ password }),
    });
    setCSRF(value.csrfToken);
  },
  async recover(recoveryKey: string, password: string): Promise<string> {
    const value = await request<{ recoveryKey: string; csrfToken: string }>("/session/recover", {
      method: "POST",
      body: JSON.stringify({ recoveryKey, password }),
    });
    setCSRF(value.csrfToken);
    return value.recoveryKey;
  },
  async lock(): Promise<void> {
    await request<void>("/session/lock", { method: "POST", body: "{}" });
    setCSRF(undefined);
  },
  async credentials(): Promise<Credential[]> {
    return (await request<{ credentials: Credential[] }>("/credentials")).credentials;
  },
  async createCredential(input: CredentialInput): Promise<Credential> {
    return request<Credential>("/credentials", { method: "POST", body: JSON.stringify(input) });
  },
  async updateCredential(id: string, input: CredentialInput): Promise<Credential> {
    return request<Credential>(`/credentials/${id}`, {
      method: "PATCH",
      body: JSON.stringify(input),
    });
  },
  deleteCredential(id: string): Promise<void> {
    return request<void>(`/credentials/${id}`, { method: "DELETE", body: "{}" });
  },
  revealSecret(id: string): Promise<SecretView> {
    return request<SecretView>(`/credentials/${id}/secret`);
  },
  codes(): Promise<CodesResponse> {
    return request<CodesResponse>("/codes");
  },
  async groups(): Promise<Group[]> {
    return (await request<{ groups: Group[] }>("/groups")).groups;
  },
  createGroup(name: string, color: string): Promise<Group> {
    return request<Group>("/groups", { method: "POST", body: JSON.stringify({ name, color }) });
  },
  deleteGroup(id: string): Promise<void> {
    return request<void>(`/groups/${id}`, { method: "DELETE", body: "{}" });
  },
  async apiKeys(): Promise<APIKey[]> {
    return (await request<{ apiKeys: APIKey[] }>("/settings/api-keys")).apiKeys;
  },
  createAPIKey(name: string): Promise<CreatedAPIKey> {
    return request<CreatedAPIKey>("/settings/api-keys", {
      method: "POST",
      body: JSON.stringify({ name }),
    });
  },
  deleteAPIKey(id: string): Promise<void> {
    return request<void>(`/settings/api-keys/${id}`, { method: "DELETE", body: "{}" });
  },
  changePassword(current: string, replacement: string): Promise<void> {
    return request<void>("/settings/password", {
      method: "POST",
      body: JSON.stringify({ current, replacement }),
    });
  },
  async rotateRecovery(): Promise<string> {
    return (
      await request<{ recoveryKey: string }>("/settings/recovery/rotate", {
        method: "POST",
        body: "{}",
      })
    ).recoveryKey;
  },
  async exportBackup(password: string): Promise<Blob> {
    const response = await fetch("/api/v1/backups/export", {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json", "X-CSRF-Token": csrfToken },
      body: JSON.stringify({ password }),
    });
    if (!response.ok) {
      const failure = (await response.json()) as { error: { code: string; message: string } };
      throw new APIError(failure.error.code, failure.error.message, response.status);
    }
    return response.blob();
  },
  previewBackup(file: File, password: string): Promise<BackupPreview> {
    const form = new FormData();
    form.set("file", file);
    form.set("password", password);
    return request<BackupPreview>("/backups/preview", { method: "POST", body: form });
  },
  applyBackup(id: string, mode: "merge" | "replace"): Promise<void> {
    return request<void>(`/backups/${id}/apply`, {
      method: "POST",
      body: JSON.stringify({ mode }),
    });
  },
};
