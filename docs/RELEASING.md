# Release runbook

This runbook covers stable Local TOTP releases. Only the maintainer may approve the release pull request, create release tags, or publish images and releases.

## Versioning and compatibility

Local TOTP follows [Semantic Versioning](https://semver.org/):

- Patch releases fix compatible defects and vulnerabilities.
- Minor releases add backward-compatible functionality.
- Major releases may change the stable interface described in [ADR 0003](adr/0003-stable-v1-interface.md).

Throughout 1.x, `/api/v1`, documented environment variables, machine-readable CLI JSON, backup v1 readability, and release artifact names remain backward compatible. Additive API fields and endpoints may ship in a minor release. Removals or incompatible semantic changes require a new API version and a major release.

SQLite is an internal implementation detail and migrates forward. Never run an older binary against a database after a storage migration. Restore a pre-upgrade snapshot or encrypted backup instead.

Deprecated interfaces are documented in the changelog and retained for at least one minor release when security and correctness permit. Breaking removals wait for the next major release.

## Prepare a release

1. Create `release/vX.Y.Z` from current `main`.
2. Update `VERSION`, both package manifests and lockfiles, OpenAPI `info.version`, the documentation examples, and `CHANGELOG.md`.
3. Run `node scripts/validate-version.mjs` and generate notices with `node scripts/generate-third-party-notices.mjs`.
4. Run every Go, frontend, site, contract, end-to-end, Docker, vulnerability, secret, licence, and generated-file check.
5. Confirm zero open critical or high dependency alerts and review every open Dependabot pull request.
6. Export an encrypted `.ltotp` backup or snapshot the test volume. Verify setup, unlock, backup export/import, CLI read, API read, and `/healthz` using release-candidate artifacts.
7. Open a pull request, resolve all conversations, record the exact green commit, and merge with squash after maintainer review.

## Publish

1. Confirm the Pages site is healthy and references the release version.
2. Create an annotated immutable tag on the recorded commit: `git tag -a vX.Y.Z <sha> -m "Local TOTP vX.Y.Z"`.
3. Push only that tag. A manual release-workflow dispatch is dry-run-only; publication occurs only for a matching `vX.Y.Z` tag.
4. Wait for all verification, binary, image, scan, SBOM, provenance, and GitHub Release jobs to succeed.
5. Make the GHCR package public if it is not already public.

The workflow publishes image tags `vX.Y.Z`, `X.Y`, `X`, and `latest`. They must resolve to the same digest. Release archives include the binary, README, LICENSE, NOTICE, and third-party notices. The release also includes checksums, an SPDX SBOM, and provenance.

## Verify after publication

- Verify every archive against `SHA256SUMS` and inspect its attestation.
- Pull the exact image anonymously by tag and digest; inspect version, source, revision, and licence labels.
- Confirm `local-totp version`, `/healthz`, new-vault setup, unlock, an encrypted backup round trip, one synthetic credential, a CLI read, and an API read.
- Confirm the documentation site, source tag, GitHub Release, image labels, and usage examples all reference the same version.
- Re-run dependency, CodeQL, Gitleaks, and container scans and preserve release artifacts and digests offline.

## Rollback and hotfixes

Never move, replace, or delete a published stable version tag to correct a release. If publication partially fails, rerun the failed job when safe or publish a patch; do not retag another commit as the same version.

For an application rollback, restore the pre-upgrade volume snapshot or encrypted backup before starting the older binary. Forward-migrated databases are not downgrade compatible.

For a hotfix, create `release/vX.Y.(Z+1)`, update every version surface and the changelog, repeat the full verification and publication process, and move only the floating container tags after the immutable patch tag succeeds.

## Backups and recovery

GitHub hosts source and release artifacts, but the maintainer also preserves release checksums, SBOMs, attestations, and image digests offline. Before any storage-affecting upgrade, verify a restorable encrypted backup or volume snapshot. Recovery exercises must use synthetic data only.
