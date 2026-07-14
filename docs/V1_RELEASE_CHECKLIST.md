# v1.0.0 release checklist

Check each item against the exact green commit recorded for the release. Publication steps are intentionally manual.

## Before publicity

- [ ] Merge Dependabot pull requests #1 and #6 only after rebasing, correcting their titles, and obtaining completely green checks.
- [ ] Leave #2, #3, #8, and #9 open with their documented follow-up; close #7 as superseded by #1.
- [ ] Merge the owner-reviewed `release/v1.0.0` pull request manually.
- [ ] Confirm Go, frontend, documentation, contract, application E2E, Docker, security, licence, CodeQL dry-run/skip, and version checks pass.
- [ ] Confirm `govulncheck`, full npm audits, full-history Gitleaks, and a critical/high container scan are clean.
- [ ] Confirm there are no open critical or high Dependabot alerts.
- [ ] Delete the obsolete `v0.1.0` GitHub Release, remote/local tag, and GHCR package version without rewriting history.
- [ ] Confirm README, site, package manifests and locks, OpenAPI, `VERSION`, and changelog all reference `1.0.0`.
- [ ] Record the exact green `main` SHA.

## Public cutover

- [ ] Change repository visibility to public.
- [ ] Immediately import and activate `Protect main` and `Protect release tags` from `.github/rulesets`.
- [ ] Configure read-only Actions tokens, SHA pinning, no Actions PR approval, verified actions, and outside-collaborator approval.
- [ ] Enable Code Security, Secret Protection and push protection, Dependabot alerts/security updates, and private vulnerability reporting.
- [ ] Run CodeQL and confirm clean Go and JavaScript/TypeScript results.
- [ ] Set the description, Pages homepage, and topics; retain only the owner as collaborator.
- [ ] Configure Pages to deploy through GitHub Actions and restrict `github-pages` to `main`.
- [ ] Deploy Pages, enforce HTTPS, and verify canonical URLs, assets, direct routes, 404 handling, sitemap, robots, metadata, accessibility, and the absence of sensitive build content.

## Release

- [ ] Create annotated tag `v1.0.0` on the recorded `main` SHA and push only that tag.
- [ ] Confirm six binary archives, checksums, SPDX SBOM, provenance, and the multi-architecture image publish successfully.
- [ ] Make the linked GHCR package public after the v1 image exists.
- [ ] Verify `v1.0.0`, `1.0`, `1`, and `latest` resolve to the same digest.
- [ ] Verify checksums and attestations and test an anonymous image pull.
- [ ] Verify `local-totp version`, `/healthz`, new-vault setup, unlock, encrypted backup import, CLI read, and API read.
- [ ] Confirm Pages, source tag, GitHub Release, image labels, and documentation all reference `1.0.0`.
- [ ] Run post-release dependency and security scans and preserve release artifacts and digests offline.

## Failure handling

- [ ] Never move or replace `v1.0.0`; ship `v1.0.1` for corrections.
- [ ] Restore a pre-upgrade volume or encrypted backup before running an older binary after a storage migration.
- [ ] If publication partially fails, rerun the failed job when safe or ship a patch; never retag another commit as `v1.0.0`.
