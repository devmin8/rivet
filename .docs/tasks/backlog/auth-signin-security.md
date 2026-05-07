# Auth signin security follow-ups

- [ ] Add signin abuse controls before a public PaaS launch.
  - Per-IP and per-username throttles.
  - Progressive backoff after repeated failures.
  - Global guardrail for credential stuffing spikes.
  - Clear behavior for when to challenge, temporarily block, or alert.

- [ ] Add security audit logging for signin.
  - Successful signin with user ID and request metadata.
  - Failed signin without logging passwords or raw secrets.
  - Inactive account attempts.
  - Session creation failures.
  - Enough structure to power alerts and incident review later.
