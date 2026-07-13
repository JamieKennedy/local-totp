# Agent Guide

Read `ARCHITECTURE.md`, `CONTRIBUTING.md`, and `SECURITY.md` before changing code.

- Preserve dependency direction: adapters -> application -> vault/totp.
- Keep SQLite inside `vault`; test with temporary real databases.
- Never log or commit secrets, databases, `.ltotp` files, current codes, or generated runtime data.
- `api/openapi.json` is canonical; regenerate `web/src/api/schema.d.ts` after interface changes.
- `internal/webui/dist` is generated and must not be committed.
- Keep shadcn primitives local under `web/src/components/ui`; use Tailwind utilities and TanStack Query/Form/Table rather than parallel state or styling systems.
- Run Go, frontend, Docker, and generated-file checks before publishing.
- ADRs are required for cryptography, storage, authentication, public-interface, and major-dependency changes.
