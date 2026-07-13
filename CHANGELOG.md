# Changelog

## [Unreleased]

### Fixed

- Allow private releases to publish when GitHub artifact attestations are unavailable, and
  include the source SBOM in release checksums.

## [0.1.0] - 2026-07-13

### Added

- Localhost-only React and TypeScript dashboard built with shadcn, Tailwind CSS, and TanStack Query, Form, and Table.
- Manual, `otpauth://` URI, QR-image, and generated-secret credential workflows.
- RFC 6238 SHA-1, SHA-256, and SHA-512 TOTP generation with configurable digits and periods.
- Encrypted SQLite vault with password and recovery-key unlock paths, groups, tags, favorites, and seed reveal.
- Encrypted backup preview, merge, and replace workflows.
- Named read-only API keys and a script-friendly server/CLI binary.
- Non-root, single-image Docker deployment with persistent `/data` storage.
- Versioned OpenAPI contract, generated TypeScript definitions, Playwright coverage, and reusable CI verification.
- Private multi-platform release automation for images, binaries, checksums, SBOMs, and provenance.

### Changed

- Adjusted browser countdowns and rollover refreshes to follow measured server time.

### Fixed

- Prevented the embedded single-page application from redirecting `/` indefinitely.
- Prevented cached code responses from being misreported as browser/server clock skew.

### Security

- Encrypts vault records with AES-256-GCM and wraps the vault key with Argon2id-derived password and recovery keys.
- Uses session-only HttpOnly SameSite cookies, CSRF protection, secret redaction, and read-only hashed API keys.
