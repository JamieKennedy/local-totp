# Local TOTP

Local TOTP is a localhost-only workbench for test and staging credentials. It keeps disposable development accounts out of your personal authenticator while providing a dashboard, read-only HTTP interface, and script-friendly CLI.

> Do not use Local TOTP for personal or production MFA credentials. Anyone with administrator access to the host or live process memory is outside the vault threat model.

## Quick start

```sh
docker build -t local-totp .
docker volume create local-totp-data
docker run --rm --name local-totp \
  -p 127.0.0.1:8080:8080 \
  -v local-totp-data:/data \
  local-totp
```

Open <http://localhost:8080>, create a master password, and save the one-time recovery key.

For unattended test environments, mount the password in a separate file:

```sh
docker run --rm --name local-totp \
  -p 127.0.0.1:8080:8080 \
  -v local-totp-data:/data \
  -v /host/path/master-password:/run/secrets/local_totp_master_password:ro \
  -e LOCAL_TOTP_MASTER_PASSWORD_FILE=/run/secrets/local_totp_master_password \
  local-totp
```

## CLI

Create a named read-only API key in Settings, place it in a user-readable file, then configure the client:

```sh
local-totp configure --url http://localhost:8080 --api-key-file /path/to/api-key
local-totp list
local-totp code "Example:developer@example.test"
local-totp codes --json
```

Run `local-totp help` for all commands. The default output of `code` is only the numeric value, suitable for scripts.

## Development

Requirements are Go 1.26.5, Node 24 LTS, npm, and Docker. See [CONTRIBUTING.md](CONTRIBUTING.md) for local commands and repository policy, [ARCHITECTURE.md](ARCHITECTURE.md) for module design, and [SECURITY.md](SECURITY.md) for the threat model.

The backend defaults to `:8080` and `./data` outside the container. During frontend development, Vite proxies `/api` and `/healthz` to the Go server.

## Releases

Release tags publish private images to `ghcr.io/jamiekennedy/local-totp` and attach cross-platform binaries, checksums, SBOMs, and attestations to a private GitHub Release. Authenticate with GitHub before pulling the private image.
