# Security Policy

## Scope

Local TOTP is designed only for synthetic test and staging credentials on a trusted developer machine. It is not a general-purpose authenticator and must not hold personal or production MFA seeds.

Encryption protects copied database files and backups. It does not protect against a host administrator, container administrator, malicious local process with equivalent privileges, browser compromise, or live process-memory inspection.

## Reporting

Report vulnerabilities privately through GitHub's private vulnerability reporting for `JamieKennedy/local-totp`. Do not open a public issue containing a seed, password, API key, recovery key, backup, or exploit details.

## Logging and fixtures

Passwords, seeds, recovery keys, API keys, `otpauth` URIs, current codes, backups, and decrypted records must never be logged. Test fixtures must be clearly synthetic; RFC vectors are the only allowlisted secret-like fixtures.

## Supported versions

Until 1.0, only the latest tagged release receives security fixes.
