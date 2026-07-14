# Local TOTP

[![CI](https://github.com/JamieKennedy/local-totp/actions/workflows/ci.yml/badge.svg)](https://github.com/JamieKennedy/local-totp/actions/workflows/ci.yml)
[![CodeQL](https://github.com/JamieKennedy/local-totp/actions/workflows/codeql.yml/badge.svg)](https://github.com/JamieKennedy/local-totp/actions/workflows/codeql.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

Local TOTP is a localhost-only workbench for synthetic test and staging TOTP credentials. It keeps disposable development accounts out of a personal authenticator while providing an encrypted dashboard, read-only HTTP v1 API, and script-friendly CLI.

> Do not use Local TOTP for personal or production MFA credentials. Host administrators, container administrators, malicious local processes, a compromised browser, and live process memory are outside the vault threat model.

## Stability

Version **1.0.0** is the first stable public release. The documented `/api/v1` interface, environment variables, machine-readable CLI JSON, backup v1 readability, and release artifact naming are backward compatible throughout 1.x. Additive changes may appear in minors; incompatible changes require a new major version. SQLite is internal and migrates forward only.

## Prerequisites

The recommended installation uses the published container image and requires Docker Engine with named volumes. The image supports Linux `amd64` and `arm64`; the web interface is tested in current Playwright Chromium, Firefox, and WebKit engines.

Release binaries are also available for Linux, macOS, and Windows on `amd64` and `arm64`. Initial setup always requires a browser.

## Install with the published container

The GHCR image is public and supports anonymous pulls. Bind the host port to loopback and use an exact version tag:

```sh
docker volume create local-totp-data
docker pull ghcr.io/jamiekennedy/local-totp:v1.0.0
docker run --detach --name local-totp \
  --restart unless-stopped \
  --publish 127.0.0.1:8080:8080 \
  --volume local-totp-data:/data \
  ghcr.io/jamiekennedy/local-totp:v1.0.0
```

Open <http://localhost:8080>, create a master password of at least 12 characters, and save the one-time recovery key separately. The image runs without a shell as UID/GID `65532`.

For Compose, use a release image rather than a local build:

```yaml
services:
  local-totp:
    image: ghcr.io/jamiekennedy/local-totp:v1.0.0
    restart: unless-stopped
    ports:
      - "127.0.0.1:8080:8080"
    volumes:
      - local-totp-data:/data

volumes:
  local-totp-data:
```

See the [installation guide](https://jamiekennedy.github.io/local-totp/docs/installation/) for attested release archives and checksums and the [deployment guide](https://jamiekennedy.github.io/local-totp/docs/deployment/) for Compose, unattended unlock, backup, and volume guidance.

## Configuration

| Variable                          | Binary default          | Container default | Purpose                                                         |
| --------------------------------- | ----------------------- | ----------------- | --------------------------------------------------------------- |
| `LOCAL_TOTP_LISTEN_ADDR`          | `127.0.0.1:8080`        | `:8080`           | HTTP listen address. Remote deployment is unsupported.          |
| `LOCAL_TOTP_DATA_DIR`             | `./data`                | `/data`           | Directory containing the encrypted SQLite file.                 |
| `LOCAL_TOTP_MASTER_PASSWORD_FILE` | unset                   | unset             | Optional file used only to unlock an existing vault at startup. |
| `LOCAL_TOTP_URL`                  | `http://localhost:8080` | —                 | CLI server URL.                                                 |
| `LOCAL_TOTP_API_KEY_FILE`         | unset                   | —                 | CLI file containing a named read-only API key.                  |

For unattended test environments, mount the password as a read-only Docker secret; never put it directly in an environment variable or Compose file. Initial vault setup remains interactive.

## CLI usage

Create a named read-only API key in **Settings**, save its one-time value in a user-readable file, then configure the release binary:

```sh
local-totp configure --url http://localhost:8080 --api-key-file /path/to/api-key
local-totp status --json
local-totp list
local-totp code "Example:developer@example.test"
local-totp codes --json
```

Bearer keys can read credential metadata, groups, and current codes. They cannot reveal seeds, unlock the vault, modify records, manage settings, or operate backups. See the [CLI reference](https://jamiekennedy.github.io/local-totp/docs/cli/).

## API usage

The server exposes the canonical contract at <http://localhost:8080/api/v1/openapi.json>. Public lifecycle status needs no key:

```sh
curl --fail http://localhost:8080/api/v1/status
```

Use a named API key for read-only automation:

```sh
curl --fail \
  --header "Authorization: Bearer $(cat /path/to/api-key)" \
  http://localhost:8080/api/v1/credentials
```

Browser management requests use an HttpOnly SameSite cookie and `X-CSRF-Token`; do not automate management by copying browser sessions. See the [HTTP API guide](https://jamiekennedy.github.io/local-totp/docs/api/) and [static OpenAPI document](https://jamiekennedy.github.io/local-totp/openapi.json).

## Upgrades and rollback

Before every upgrade, export an encrypted `.ltotp` backup or stop the container and snapshot its volume. Pull a new exact tag, update the Compose image, and recreate the container with the existing volume.

Never run an older binary against a database after a forward migration. Roll back by restoring the pre-upgrade volume or an encrypted backup readable by the older version. Exact tags such as `v1.0.0` are immutable; fixes are released as patches such as `v1.0.1`. Moving tags (`1`, `1.0`, `latest`) may advance only after a successful release.

## Troubleshooting

- Check `curl --fail http://localhost:8080/healthz` and `docker logs local-totp` without posting sensitive output publicly.
- Confirm the port mapping begins with `127.0.0.1:` and that no other process owns the host port.
- A locked vault after restart is expected unless an existing vault is unlocked interactively or through a mounted password file.
- Bind mounts must be writable by UID/GID `65532`; a named Docker volume is recommended.
- API `401` responses mean the session or bearer key is invalid; `403 csrf_failed` means a browser write omitted the current CSRF token; `423 vault_locked` means the in-memory vault key is unavailable.

Use only synthetic examples in support requests. See [SUPPORT.md](SUPPORT.md) for support expectations.

## Security and privacy

Local TOTP sends no telemetry and has no cloud service. Passwords, seeds, recovery keys, API keys, current codes, backups, and decrypted records must never be logged or committed. Report vulnerabilities privately through the process in [SECURITY.md](SECURITY.md); the maintainer targets acknowledgement within seven days on a best-effort basis.

## Contributing and development

The primary installation path is the published release image above. Source builds are for contributors: read [CONTRIBUTING.md](CONTRIBUTING.md) before changing code, [ARCHITECTURE.md](ARCHITECTURE.md) for module boundaries, [GOVERNANCE.md](GOVERNANCE.md) for ownership, and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) for community standards.

## Licence and release integrity

Local TOTP is licensed under [Apache-2.0](LICENSE), including an explicit patent grant. [NOTICE](NOTICE) and [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) accompany source, binary, container, and documentation distributions. Releases include SHA-256 checksums, an SPDX source SBOM, OCI SBOM/provenance, and GitHub attestations.
