# Local TOTP v1.0.0

Local TOTP is a localhost-only workbench for synthetic test and staging TOTP credentials. It is not a personal authenticator and must not store production MFA seeds.

## Highlights

- Encrypted SQLite vault using AES-256-GCM with Argon2id password wrapping and a separately protected recovery path.
- React dashboard for importing, generating, grouping, backing up, and inspecting synthetic credentials.
- Read-only HTTP v1 API and script-friendly CLI with named hashed API keys.
- Non-root `ghcr.io/jamiekennedy/local-totp:v1.0.0` image for Linux amd64 and arm64.
- Release binaries for Linux, macOS, and Windows on amd64 and arm64.
- Public documentation, OpenAPI contract, checksums, SPDX SBOM, container SBOM/provenance, and release attestations.

## Security boundary

- Bind deployments to loopback and use only trusted developer machines.
- Encryption protects copied files and backups, not a compromised host, browser, container administrator, or live process memory.
- Current codes, seeds, passwords, recovery keys, API keys, and backups must not be logged or committed.

## Upgrade from the private preview

1. Export an encrypted `.ltotp` backup or snapshot the existing volume.
2. Pull the exact `v1.0.0` image.
3. Recreate the container with the existing `/data` volume and loopback port binding.
4. Verify `/healthz`, unlock the vault, and test one synthetic credential and CLI/API read.

## Known limitations

- Single-user, single-process, localhost-only operation.
- No remote hosting, cloud synchronization, high availability, or production-MFA support.
- Storage migrations are forward-only; downgrade by restoring a pre-upgrade backup.
