# Architecture

Local TOTP is a single-process system with an embedded React interface, a versioned HTTP interface, a CLI adapter, and one SQLite file.

## Module map

- `internal/totp` is a pure module. Its interface accepts explicit time and owns Base32 normalization, `otpauth` parsing/building, validation, and RFC 6238 generation.
- `internal/vault` is a deep module. Its interface hides encryption, key wrappers, migrations, SQLite transactions, credential/group persistence, API-key hashes, and backup serialization.
- `internal/application` owns workflows and coordinates `vault` with `totp`.
- `internal/httpapi` and `internal/cli` are adapters. They validate transport input and translate results; they do not own business rules.
- `web/src/features` contains feature modules. `web/src/api` is the only frontend module allowed to perform HTTP requests. TanStack Query owns server state and invalidation, TanStack Form owns form state/validation, and TanStack Table owns the credential grid. shadcn components copied under `web/src/components/ui` form the local interface seam over Radix and Tailwind CSS.
- `site` is the independent static Astro landing and documentation site. It has no runtime dependency on the application, keeps its own shadcn primitives, and publishes the canonical `api/openapi.json` at build time without duplicating its source.

Dependency direction is `cmd -> adapters -> application -> vault/totp`. Lower modules never import adapters.

SQLite is local-substitutable and tested through temporary real databases. There is no repository interface or ORM. Add an interface only at a real seam where behavior varies.

## Security data flow

The master password is passed to Argon2id and used only to unwrap a random data-encryption key. A separately derived recovery-key wrapper protects the same key. Credential and group JSON records are encrypted independently with AES-256-GCM; record ID and schema version are authenticated as additional data.

The React interface never receives a seed except through the explicit seed-reveal route. Read-only API keys can retrieve metadata and current codes, but never seeds or management functions. Sessions and the unwrapped data key exist only in memory and are invalidated on lock or process restart.

## Decisions

Add an ADR under `docs/adr` before changing cryptography, storage format, authentication, the public HTTP interface, or a major runtime dependency. Ordinary implementation details do not need ADRs.
