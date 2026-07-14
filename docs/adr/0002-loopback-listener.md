# ADR 0002: Loopback listener by default

- Status: Accepted
- Date: 2026-07-14

## Context

Local TOTP is a single-user developer tool for synthetic credentials. The HTTP adapter rejects non-loopback `Host` and `Origin` values, but those headers are request data rather than a network security boundary. The standalone binary previously listened on every interface through the `:8080` default, which could expose the service to the local network despite the header checks.

The container needs to listen on all container interfaces so Docker can forward a host-loopback port into it.

## Decision

The standalone `local-totp serve` command defaults `LOCAL_TOTP_LISTEN_ADDR` to `127.0.0.1:8080`. An operator may explicitly override the address, but remote deployment is outside the supported threat model. The production container continues to set `LOCAL_TOTP_LISTEN_ADDR=:8080`; documented Docker and Compose examples publish it only through `127.0.0.1` on the host.

Host and Origin validation remains defence in depth. It must not be described as a substitute for loopback binding.

## Consequences

- A default standalone launch is unreachable from other machines.
- Container networking continues to work with the existing loopback port mapping.
- Users choosing another listen address accept an unsupported configuration and must provide their own access-control boundary.
