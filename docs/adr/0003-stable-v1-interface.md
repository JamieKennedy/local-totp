# ADR 0003: Stable v1 public interface

- Status: Accepted
- Date: 2026-07-14

## Context

Local TOTP 1.0 publishes an HTTP API, environment variables, machine-readable CLI output, encrypted backup format, and named release artifacts. Automation and upgrade guidance need a clear compatibility boundary before these surfaces are declared stable.

SQLite tables are an internal implementation detail. Storage migrations are forward-only, so database-file compatibility cannot safely promise downgrade support.

## Decision

Throughout the 1.x series:

- documented `/api/v1` operations and authentication semantics remain backward compatible;
- documented `LOCAL_TOTP_*` environment variables retain their meanings;
- documented machine-readable CLI JSON fields remain backward compatible;
- backup format version 1 remains readable; and
- release archive and container tag naming remain stable.

Minor releases may add optional request fields, response fields, endpoints, enum values where callers are required to tolerate extension, and CLI commands. Removing a documented field or operation, changing authentication requirements incompatibly, or changing an existing field's meaning requires a new API version and a major release.

SQLite is internal and may migrate forward during startup. Running an older binary against a database after a storage migration is unsupported. Downgrade requires restoring a pre-upgrade volume snapshot or an encrypted backup that the older version can read.

The canonical contract is `api/openapi.json`; generated TypeScript declarations and the published documentation copy must match it exactly.

## Consequences

- Patch and minor releases must validate all versioned surfaces and compatibility tests before publication.
- Clients should ignore unknown response fields and handle documented error codes.
- Breaking interface work requires an ADR, a major release, and an explicit migration path.
- Backups and volume snapshots are required rollback tools; the live SQLite file is not a portable public API.
