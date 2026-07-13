import { afterEach, describe, expect, it, vi } from "vitest";
import { api, setCSRF } from "./client";

afterEach(() => {
  setCSRF(undefined);
  vi.unstubAllGlobals();
});

describe("API client", () => {
  it("captures the session CSRF token and sends it on mutations", async () => {
    const fetchMock = vi.fn<typeof fetch>();
    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            setup: true,
            locked: false,
            authenticated: true,
            csrfToken: "test-csrf",
            version: "test",
            testOnly: true,
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      )
      .mockResolvedValueOnce(new Response(undefined, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);

    await api.status();
    await api.deleteCredential("credential-id");

    const mutation = fetchMock.mock.calls[1];
    expect(mutation?.[0]).toBe("/api/v1/credentials/credential-id");
    expect(new Headers(mutation?.[1]?.headers).get("X-CSRF-Token")).toBe("test-csrf");
  });

  it("normalizes structured API errors", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValue(
        new Response(
          JSON.stringify({ error: { code: "vault_locked", message: "Unlock the vault" } }),
          { status: 423, headers: { "Content-Type": "application/json" } },
        ),
      );
    vi.stubGlobal("fetch", fetchMock);

    const request = api.credentials();
    await expect(request).rejects.toMatchObject({
      code: "vault_locked",
      message: "Unlock the vault",
      status: 423,
    });
  });
});
