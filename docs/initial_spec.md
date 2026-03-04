# WHOOPY Go Rewrite – Initial Specification

## 1. Project Overview
- Replace the existing Bun/TypeScript `whoop-cli` with a cross-platform Go CLI (working name: `whoopy`).
- Use WHOOP’s **official OAuth 2.0** developer platform instead of reverse-engineered mobile endpoints.
- Primary goals: one-time OAuth login with refresh-token persistence, stable JSON/text outputs for automation, and feature parity (plus expansion) with the current CLI.
- Current status (Mar 4, 2026): Go module initialized with Cobra-powered root command, config + token storage packages in place, and CLI UX principles recorded. Next focus: OAuth PKCE implementation.

## 2. Guiding Objectives
1. **Credential safety** – never store raw passwords; rely on WHOOP-issued refresh tokens obtained via the Authorization Code flow with `offline` scope.
2. **Agent-friendly UX** – commands should default to deterministic JSON while offering human-readable text for quick inspection.
3. **Cross-platform binaries** – target macOS (arm64 + amd64), Linux (amd64 + arm64), and Windows (amd64) with release artifacts and optional Homebrew formula updates.
4. **Resilience** – automatic token refresh, structured errors, and graceful handling of WHOOP rate limits or pagination.

## 3. Authentication & Token Lifecycle
### 3.1 Authorization Flow
1. `whoopy auth login` starts an Authorization Code with PKCE flow.
2. CLI opens the default browser pointing to WHOOP’s `/oauth/oauth2/auth` endpoint with scopes `offline read:profile read:body_measurement read:cycles read:recovery read:sleep read:workout` (adjust as needed).
3. CLI spins up a temporary localhost callback (preferred) to capture the redirect; provide a fallback copy-paste code path for headless environments.
4. Exchange the returned code for access + refresh tokens via `/oauth/oauth2/token`.

### 3.2 Token Storage
- Store tokens in `$XDG_CONFIG_HOME/whoopy/tokens.json` (or `%APPDATA%\\whoopy\\tokens.json` on Windows) with `0600` permissions.
- Optionally integrate with platform keychains (macOS Keychain, Windows Credential Manager, `libsecret`) when available.
- Persist metadata: `access_token`, `refresh_token`, `expires_at`, `scope`, `token_type`.

### 3.3 Refresh Strategy
- Before each API call: if `now >= expires_at - 60s`, call `/oauth/oauth2/token` with `grant_type=refresh_token`.
- Replace both access and refresh tokens atomically in storage (WHOOP rotates refresh tokens).
- Log minimal info on disk; prefer structured debug logs via `whoopy --debug`.

### 3.4 Logout & Revocation
- `whoopy auth logout` calls WHOOP’s revocation endpoint (if available) or deletes cached tokens and instructs the user to revoke access from their WHOOP account.
- Ensure logout wipes local config directories securely.

### 3.5 Implementation Status (Mar 4, 2026)
- `whoopy auth login` implemented with PKCE + Authorization Code flow, local callback server on configurable `redirect_uri`, and flags for `--no-browser`, `--manual`, and `--code` (paste redirect URL manually).
- `whoopy auth status` reports stored token scopes + expiry; `whoopy auth logout` clears tokens and best-effort revokes refresh tokens.
- Config now exposes `oauth_base_url` and `redirect_uri` fields (env overrides: `WHOOPY_OAUTH_BASE_URL`, `WHOOPY_REDIRECT_URI`).
- Running `whoopy auth login` scaffolds `~/.config/whoopy/config.toml` with sample values if missing and instructs the user to edit it.
- Foundational unit tests in place for config loading, token storage, and OAuth flow helpers (PKCE generation, token exchange/refresh/logout).
- Core WHOOP API client implemented (token injection, auto-refresh, 401 retry, 429 backoff, JSON helper) to power upcoming commands.
- `whoopy profile show` implemented (JSON by default, `--text` for human-readable) fetching `/user/profile/basic` and `/user/measurement/body`.
- `whoopy workouts list` implemented with shared pagination flags, default JSON output (`workouts` array + `next_token`), and `--text` tables showing start time, duration, sport, strain, and avg HR. Client-side filters `--sport`, `--min-strain`, and `--max-strain` narrow the displayed workouts without changing server pagination. Data comes from `GET /developer/v2/workout`, pulling the documented score metrics (strain, HR, distances, zone durations) exposed by WHOOP’s official schema. Reference: https://developer.whoop.com/api#tag/Workout/operation/getWorkoutCollection.
- `whoopy workouts view <id>` implemented to call `GET /developer/v2/workout/{id}` and render either the raw JSON object or a detailed text summary (start/duration, state, strain, HR, kJ, distance, percent recorded, zone splits). This shares the same service + formatting helpers as the list command so future consumers get consistent schemas across both commands.
- `whoopy cycles list` / `whoopy cycles view <id>` implemented against `GET /developer/v2/cycle` and `GET /developer/v2/cycle/{id}`. Lists honor the shared pagination flags plus `--text` output, showing start date, duration, strain, and heart-rate metrics. Detail view surfaces timestamps, strain, kJ, heart rates, and timezone offset in both JSON and text formats. Reference: https://developer.whoop.com/api#tag/Cycle/operation/getCycleCollection.
- `whoopy recovery list` / `whoopy recovery view <cycle-id>` implemented using `GET /developer/v2/recovery` plus `GET /developer/v2/cycle/{cycle_id}/recovery`. Lists highlight cycle IDs, recovery score, resting HR, HRV, respiratory rate, calibration flag, and sleep IDs; the detail command dumps the full physiology profile (SpO₂, skin temp, strain). Reference: https://developer.whoop.com/api#tag/Recovery/operation/getRecoveryCollection.
- `whoopy sleep list` / `whoopy sleep view <sleep-id>` implemented via `GET /developer/v2/activity/sleep` and `GET /developer/v2/activity/sleep/{sleep_id}`. Lists show local start time, duration, performance %, respiratory rate, nap flag, and ID; detailed view breaks down efficiency, consistency, and per-stage durations. Reference: https://developer.whoop.com/api#tag/Sleep/operation/getSleepCollection.
- `whoopy stats daily --date YYYY-MM-DD [--text]` implemented atop a new `internal/stats` service that aggregates cycles, recovery, sleep, and workouts for a calendar day. JSON output includes the raw resources plus a summary block (cycle strain, recovery score, sleep performance, total sleep hours, workout count/strain); text mode renders a multi-section dashboard. The service reuses the official developer endpoints listed above and respects the shared pagination helpers.
- Convenience `today` subcommands for workouts, recovery, and sleep call a shared helper that locks the range to the current local calendar day (midnight-to-midnight UTC-converted) with a default limit of 25. This gives users a zero-config snapshot via `whoopy <feature> today [--text]`.
- `whoopy workouts export` streams workouts over any range as either JSON Lines (default) or CSV, auto-paginating via `next_token` and applying the same sport/strain filters as `workouts list`. Users can send output to stdout or `--output <path>` for scripts.
- `whoopy diag [--text]` now prints config + token file locations, credential presence, token freshness, and a lightweight `/user/profile/basic` probe so users can quickly see whether they're authenticated and whether the API is reachable.
- Build metadata (`whoopy version` / `whoopy --version`) follows stamp’s ldflag strategy so tagged releases surface SemVer, commit SHA, and build date.

## 4. Configuration & Environment
- Require WHOOP-issued **client ID** and **client secret** (if confidential client). Support reading from:
  - `WHOOPY_CLIENT_ID` / `WHOOPY_CLIENT_SECRET`
  - `$XDG_CONFIG_HOME/whoopy/config.toml`
- Allow overriding base API URL for testing.
- Additional overrides: `WHOOPY_OAUTH_BASE_URL` (default `https://api.prod.whoop.com/oauth`) and `WHOOPY_REDIRECT_URI` (default `http://127.0.0.1:8735/oauth/callback`).
- Global flags: `--json` (default), `--text`, `--pretty`, `--debug`, `--config <path>`.

## 5. Planned Feature Set
| Feature | Endpoints | Notes |
| --- | --- | --- |
| Profile summary | `GET /developer/v2/user/profile/basic`, `GET /developer/v2/user/measurement/body` | Show name, email, locale, height/weight, max HR. |
| Cycle listing | `GET /developer/v2/cycle` (with `limit`, `start`, `end`) | Surfaces daily strain, kJ, avg/max HR, timezone. |
| Cycle detail | `GET /developer/v2/cycle/{id}` | Deep dive for one day; include workout IDs. |
| Recovery | `GET /developer/v2/recovery` | Display score, resting HR, HRV, respiratory rate. |
| Sleep | `GET /developer/v2/sleep` | Show performance %, stage durations, time in bed, respiratory rate. |
| Workouts | `GET /developer/v2/workout`, `GET /developer/v2/workout/{id}` | Include sport type, strain, zone durations, distance, calories. |
| Workouts export | `GET /developer/v2/workout` (auto-paginated) | Dump workouts as JSONL or CSV with client-side sport/strain filters for downstream analytics. |
| Diagnostics | none (local + lightweight `/user/profile/basic` probe) | Surface config/token paths, token freshness, and API connectivity via `whoopy diag`. |
| Daily stats summary | Combination of Cycle + Recovery + Sleep + Workouts for a date | Replicates existing `whoop stats` output with official data. |
| Webhook helper (stretch) | n/a (polling utility) | CLI subcommand to verify webhook payload handling by developers. |

## 6. CLI Command Sketch
```
whoopy auth login [--no-browser]
whoopy auth status
whoopy auth logout

whoopy profile show [--text]

whoopy cycles list [--start YYYY-MM-DD] [--end YYYY-MM-DD] [--limit N] [--cursor TOKEN]
whoopy cycles view <cycle-id>

whoopy recovery list [--start --end] [--limit N] [--cursor TOKEN]
whoopy recovery today
whoopy recovery view <cycle-id>

whoopy sleep list [--start --end] [--limit N] [--cursor TOKEN]
whoopy sleep today
whoopy sleep view <sleep-id>

whoopy workouts list [--start --end] [--limit N] [--sport NAME|ID] [--min-strain F] [--max-strain F]
whoopy workouts today [--sport NAME|ID] [--min-strain F] [--max-strain F]
whoopy workouts view <workout-id>
whoopy workouts export [--start --end] [--limit N] [--cursor TOKEN] [--sport NAME|ID] [--min-strain F] [--max-strain F] [--format jsonl|csv] [--output PATH|-]

whoopy diag [--text]

whoopy stats daily --date YYYY-MM-DD [--text|--json]
```
- Every `list` command paginates automatically; support `--cursor <token>` for manual control.
- `--text` renders aligned tables; JSON output includes metadata (`source`, `generated_at`, `pagination`).

### Shared Collection Plumbing (2026-03-04)
- Added `internal/api.ListOptions` to centralize WHOOP collection parameters (`start`, `end`, `limit`, `nextToken`). Helpers validate ranges, format timestamps as RFC 3339, and attach values to `url.Values` before every request. This matches the official pagination contract (`nextToken` query param + `next_token` response field) described in https://developer.whoop.com/docs/developing/overview#tag/Workout/operation/getWorkoutCollection.
- Created a generic `internal/api.Page[T]` struct with `records` + `next_token` to decode all collection endpoints consistently while exposing a `HasNext()` helper for auto-pagination loops.
- Introduced `internal/cli/addListFlags` + `parseListOptions` so every list command automatically advertises `--start`, `--end`, `--limit`, and `--cursor` flags, aligning with clig.dev guidance about predictable flag shapes. `--start/--end` accept RFC 3339 or `YYYY-MM-DD` (interpreted as UTC midnight) and pipe into `api.ListOptions`.
- Flag parsing emits user-friendly errors when the range is inverted or when the limit is negative, preventing wasted WHOOP API calls and ensuring commands fail fast before any network traffic.
- Added `todayRangeOptions(limit)` helper shared by the new `today` subcommands to encapsulate “local midnight to midnight” math and clamp the limit to a safe default.
- The workouts export implementation keeps calling `GET /developer/v2/workout` while WHOOP returns `next_token`, streaming each page either as JSON Lines or CSV. Export honors the same client-side sport/strain filters and can target stdout or `--output` paths for automation.

## 7. CLI UX Principles
- Follow the recommendations in https://clig.dev/:
  - Provide concise `whoopy --help` plus per-command help with examples.
  - Prefer subcommands over flags for distinct actions (auth/profile/cycles).
  - Support piping/automation by defaulting to JSON and avoiding noisy stdout on success.
  - Return non-zero exit codes for error conditions and send diagnostics to stderr.
  - Keep commands discoverable with `whoopy help <subcommand>` and markdown docs mirroring CLI help text.

## 8. Data Modeling & Output Contracts
- Define Go structs mirroring WHOOP v2 schemas; add adapters that map WHOOP fields to CLI-friendly names (e.g., `respiratory_rate` → `respRate`).
- Stable JSON schema examples should be stored under `docs/examples/*.json`.
- Include `source_endpoint` metadata for traceability.
- Diagnostics output (`whoopy diag`) should expose three clearly delimited blocks:
  - `config`: absolute path, existence flag, whether client ID/secret/env overrides are set, and the effective API/OAuth/redirect URLs. Do not fail if the file is missing—report the error inline.
  - `tokens`: token file path + existence, last modified timestamp, scopes, expiry timestamp, humanized remaining lifetime, and whether a refresh token is stored.
  - `api`: most recent health probe status (`ok` or `error`), latency in milliseconds, and any error message when authentication/config is incomplete. The probe can hit `/user/profile/basic` via the shared API client.

## 9. Error Handling
- Distinguish between auth errors (token missing/expired/revoked), validation errors (bad flags), and API errors (HTTP >= 400).
- Retry policy: exponential backoff on `429` up to 3 attempts; immediate fail on `401` after one refresh attempt.
- Provide `whoopy diag` to print config path, token age, and last API status for support.

## 10. Build & Distribution
- Use Go 1.22+ modules; `make build|test|install` remain the local dev loop.
- GoReleaser (`.goreleaser.yml`) builds macOS+Linux (amd64/arm64), injects `main.version/commit/date` ldflags, creates universal macOS binaries, and uploads tarballs + checksums to GitHub Releases.
- GitHub Actions workflow `.github/workflows/release.yml` mirrors stamp: trigger on `v*` tags, run `go test ./...`, then invoke GoReleaser. Attach artifacts even if the Homebrew tap token is missing.
- Homebrew-only distribution for now. The GoReleaser `brews` stanza pushes updates to `totocaster/homebrew-tap/Formula/whoopy.rb` using the `HOMEBREW_TAP_TOKEN` secret. `Formula/whoopy.rb` in this repo is a template and reference for that tap.
- Release ritual documented in `RELEASE_SETUP.md`: ensure main is green, tag `vX.Y.Z`, push tag, confirm release assets + tap update, and test `brew install whoopy`.
- Future: consider Windows builds or additional package managers once macOS/Linux are stable.

## 11. Open Questions / Follow-Ups
1. Confirm WHOOP client credentials availability (public vs confidential app).
2. Decide on datastore for tokens (file vs OS keychain abstraction).
3. Determine whether webhook helper is in scope for v1.
4. Choose output schema versioning strategy (e.g., include `schema_version` field).

## 12. Reference Links
- WHOOP Developer OAuth Overview: https://developer.whoop.com/docs/developing/oauth
- WHOOP v2 REST API Reference (profile, cycles, recovery, sleep, workouts): https://developer.whoop.com/api
- Tutorial – Fetch current recovery score: https://developer.whoop.com/docs/tutorials/get-current-recovery-score
- Webhooks Guide: https://developer.whoop.com/docs/developing/webhooks
- `getWorkoutCollection` endpoint details: https://developer.whoop.com/docs/developing/overview#tag/Workout/operation/getWorkoutCollection
- CLI design best practices: https://clig.dev/

This document should give future coding agents the context needed to start implementing the Go rewrite with the desired OAuth-first architecture and feature set. Update as requirements evolve.
