# Changelog

## [Unreleased]

No changes yet.

## [1.0.0] - 2026-07-14

### Added

- Localhost-only React and TypeScript dashboard built with shadcn, Tailwind CSS, and TanStack Query, Form, and Table.
- Manual, `otpauth://` URI, QR-image, and generated-secret credential workflows.
- RFC 6238 SHA-1, SHA-256, and SHA-512 TOTP generation with configurable digits and periods.
- Encrypted SQLite vault with password and recovery-key unlock paths, groups, tags, favorites, and seed reveal.
- Encrypted backup preview, merge, and replace workflows.
- Named read-only API keys and a script-friendly server/CLI binary.
- Non-root, single-image Docker deployment with persistent `/data` storage.
- Versioned OpenAPI contract, generated TypeScript definitions, Playwright coverage, and reusable CI verification.
- Public documentation and an OpenAPI v1 contract with generated TypeScript definitions.
- Multi-platform release automation for images, binaries, checksums, SPDX SBOMs, and provenance.
- Apache-2.0 licensing, third-party notices, community policies, and a public release runbook.

### Changed

- Adjusted browser countdowns and rollover refreshes to follow measured server time.
- Bound standalone installations to `127.0.0.1:8080` by default; containers retain their internal `:8080` listener and require an explicit loopback host mapping.
- Adopted the official GitHub Pages artifact deployment flow and hardened CI permissions, dependency checks, static analysis, and image scanning.
- Upgraded security-sensitive Go, Vite, Vitest, and Playwright dependencies within their current major versions.

### Fixed

- Prevented the embedded single-page application from redirecting `/` indefinitely.
- Prevented cached code responses from being misreported as browser/server clock skew.
- Completed management request and response schemas for the stable `/api/v1` contract.
- Added real-database HTTP security integration tests and cross-browser documentation smoke and accessibility tests.

### Security

- Encrypts vault records with AES-256-GCM and wraps the vault key with Argon2id-derived password and recovery keys.
- Uses session-only HttpOnly SameSite cookies, CSRF protection, secret redaction, and read-only hashed API keys.
- Validates Host and Origin headers, limits authentication attempts, scans release images, and verifies repository history for committed secrets.
