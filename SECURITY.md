# Security Policy

## Scope

Local TOTP is designed only for synthetic test and staging credentials on a trusted developer machine. It is not a general-purpose authenticator and must not hold personal or production MFA seeds.

Encryption protects copied database files and backups. It does not protect against a host administrator, container administrator, malicious local process with equivalent privileges, browser compromise, or live process-memory inspection.

## Reporting

Report vulnerabilities privately through [GitHub private vulnerability reporting](https://github.com/JamieKennedy/local-totp/security/advisories/new). Do not open a public issue containing a seed, password, API key, recovery key, backup, or exploit details.

The maintainer aims to acknowledge a complete report within seven days. This is a best-effort target, not a service-level agreement. Coordinated disclosure timing will be agreed with the reporter after triage.

## Logging and fixtures

Passwords, seeds, recovery keys, API keys, `otpauth` URIs, current codes, backups, and decrypted records must never be logged. Test fixtures must be clearly synthetic; RFC vectors are the only allowlisted secret-like fixtures.

## Supported versions

The latest `1.x` release receives security fixes. Older exact releases may be unsupported after a newer patch is available.

Security updates are released as new immutable patch versions. Exact release tags are never moved.
