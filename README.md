# Local TOTP

Local TOTP is a localhost-only workbench for test and staging credentials. It keeps disposable development accounts out of your personal authenticator while providing a dashboard, read-only HTTP interface, and script-friendly CLI.

> Do not use Local TOTP for personal or production MFA credentials. Anyone with administrator access to the host or live process memory is outside the vault threat model.

## Quick start

```sh
docker login ghcr.io --username <GITHUB_USERNAME>
docker volume create local-totp-data
docker run --detach --name local-totp \
  --restart unless-stopped \
  -p 127.0.0.1:8080:8080 \
  -v local-totp-data:/data \
  ghcr.io/jamiekennedy/local-totp:v0.1.0
```

Open <http://localhost:8080>, create a master password, and save the one-time recovery key.

The repository and release package are currently private. Authenticate with an account or token that can read the GHCR package. See the [installation](https://jamiekennedy.github.io/local-totp/docs/installation/) and [deployment](https://jamiekennedy.github.io/local-totp/docs/deployment/) guides for release binaries, checksum verification, version placeholders, and Docker Compose.

For unattended test environments, mount the password in a separate file:

```sh
docker run --detach --name local-totp \
  --restart unless-stopped \
  -p 127.0.0.1:8080:8080 \
  -v local-totp-data:/data \
  -v /host/path/master-password:/run/secrets/local_totp_master_password:ro \
  -e LOCAL_TOTP_MASTER_PASSWORD_FILE=/run/secrets/local_totp_master_password \
  ghcr.io/jamiekennedy/local-totp:v0.1.0
```

The password file unlocks an existing vault; initial setup still happens in the browser.

## CLI

Create a named read-only API key in Settings, place it in a user-readable file, then configure the client:

```sh
local-totp configure --url http://localhost:8080 --api-key-file /path/to/api-key
local-totp list
local-totp code "Example:developer@example.test"
local-totp codes --json
```

Run `local-totp help` for all commands. The default output of `code` is only the numeric value, suitable for scripts.

Full guides are available on the [Local TOTP documentation site](https://jamiekennedy.github.io/local-totp/):

- [Installation and setup](https://jamiekennedy.github.io/local-totp/docs/installation/)
- [Container deployment](https://jamiekennedy.github.io/local-totp/docs/deployment/)
- [CLI reference](https://jamiekennedy.github.io/local-totp/docs/cli/)
- [HTTP API reference](https://jamiekennedy.github.io/local-totp/docs/api/)

## Development

Requirements are Go 1.26.5, Node 24 LTS, npm, and Docker. See [CONTRIBUTING.md](CONTRIBUTING.md) for local commands and repository policy, [ARCHITECTURE.md](ARCHITECTURE.md) for module design, and [SECURITY.md](SECURITY.md) for the threat model.

The backend defaults to `:8080` and `./data` outside the container. During frontend development, Vite proxies `/api` and `/healthz` to the Go server.

### Documentation site

The fully static Astro site lives under `site/` and uses the Node 24 toolchain:

```sh
npm --prefix site ci
npm --prefix site run dev
npm --prefix site run verify
npm --prefix site run preview
```

`npm --prefix site run build` writes ignored output to `site/dist`. Pushes to `main` that change the site or canonical OpenAPI document publish that output to the generated `gh-pages` branch. After the first deployment creates the branch, configure repository **Settings → Pages** to deploy from `gh-pages` at `/ (root)`, then rerun the workflow. Never commit `site/dist` to `main`.

## Releases

Release tags publish private images to `ghcr.io/jamiekennedy/local-totp` and attach cross-platform binaries, checksums, SBOMs, and attestations to a private GitHub Release. Authenticate with GitHub before pulling the private image.
