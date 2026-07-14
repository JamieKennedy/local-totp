# ADR 0001: Astro documentation site

- Status: Accepted
- Date: 2026-07-14

## Context

Local TOTP needs a public-facing landing page and maintainable installation, deployment, CLI, and HTTP API documentation. The site must be fully static, work under the `/local-totp` GitHub Pages repository path, preserve the repository's React, Tailwind, TypeScript, and local-shadcn conventions, and avoid adding client-side state where static HTML is sufficient.

The existing `web/` package is the embedded application interface. Reusing it as a documentation build would couple release UI assets to publishing concerns and would make GitHub Pages routing harder to reason about.

## Decision

Create an independent `site/` package using Astro static output, strict TypeScript, MDX, the official React integration, and Tailwind CSS. Keep shadcn primitives local to the site. Astro renders content and static React components at build time; only the mobile sheet, installation tabs, and troubleshooting accordion are hydrated.

Set the canonical site origin to `https://jamiekennedy.github.io` and the Astro base path to `/local-totp`. Generate the public OpenAPI document directly from the canonical root `api/openapi.json` during the site build. Upload `site/dist` as an official GitHub Pages artifact from `main` and deploy it through the protected `github-pages` environment; generated output is never committed to a source or deployment branch.

## Consequences

- Documentation has its own npm dependency and verification lifecycle.
- The embedded application and documentation site can evolve independently while retaining shared visual conventions.
- Most pages ship static HTML and CSS; React runtime code is limited to explicitly interactive components.
- All internal URLs and assets must remain base-path-aware, and CI validates the built output against `/local-totp`.
- Pages requires a one-time repository setting selecting GitHub Actions as the source and restricting the `github-pages` environment to `main`.
