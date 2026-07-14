# Contributing

## Toolchain

- Go 1.26.5
- Node.js 24 LTS and npm for both `web/` and `site/`
- Docker with BuildKit

Install frontend dependencies with `npm --prefix web ci`. Keep `go.sum` and `web/package-lock.json` committed.
Install documentation-site dependencies with `npm --prefix site ci`. Keep `site/package-lock.json` committed.

## Verification

Before opening a pull request, run:

```sh
gofmt -w ./cmd ./internal
go vet ./...
go test -race -coverprofile=coverage-go.out ./...
npm --prefix web run format:check
npm --prefix web run lint
npm --prefix web run typecheck
npm --prefix web run test:coverage
npm --prefix web run build
npm --prefix site run format:check
npm --prefix site run lint
npm --prefix site run typecheck
npm --prefix site run build
npm --prefix site run check:dist
docker build -t local-totp:verify .
```

CI also runs static analysis, vulnerability checks, secret scanning, generated-interface verification, Playwright, and a production image build. Coverage is reported, not numerically gated. Changed behavior requires tests.

## Coding standards

Go code uses `gofmt`, `goimports`, explicit contexts for I/O, wrapped/typed errors, parameterized SQL, and structured redacted logging. Expected runtime failures never panic. Keep interfaces at real seams and prefer private functions inside deep modules.

TypeScript uses strict mode, named exports, functional React, shadcn components, and Tailwind CSS. The application uses typed calls through `web/src/api`, TanStack Query for server state, TanStack Form for forms, and TanStack Table for data grids. The static documentation site keeps its shadcn primitives local under `site/src/components/ui` and hydrates only interactive islands. Do not use `any`, default exports in application source, unhandled promises, or browser persistence for secrets.

## Branches and commits

After the initial bootstrap, branch from `main` using `feat/`, `fix/`, `refactor/`, `test/`, `docs/`, `chore/`, `ci/`, `release/`, or `agent/` plus a kebab-case description.

Pull request titles and squash commits use:

```text
type(scope)!: imperative summary
```

Allowed types are `feat`, `fix`, `docs`, `refactor`, `test`, `build`, `ci`, `chore`, `perf`, and `revert`. Keep titles under 72 characters. Use `BREAKING CHANGE:` in the body for incompatible changes. Local WIP commits may be informal because GitHub squash-merges the PR.

`main` must remain releasable. Required checks must pass; force-pushes, deletion, merge commits, and rebase merges are prohibited. A solo maintainer does not require a self-review.

## Releases

Create `release/vX.Y.Z`, update `VERSION` and `CHANGELOG.md`, and open `chore(release): vX.Y.Z`. After merge and green CI, create an annotated `vX.Y.Z` tag on the exact main commit. Never replace an existing exact version; ship a patch.
