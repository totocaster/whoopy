# Whoopy CLI

[![Release](https://img.shields.io/github/v/release/totocaster/whoopy)](https://github.com/totocaster/whoopy/releases)
[![Build](https://github.com/totocaster/whoopy/actions/workflows/release.yml/badge.svg)](https://github.com/totocaster/whoopy/actions/workflows/release.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/totocaster/whoopy)](https://go.dev/)
[![License](https://img.shields.io/github/license/totocaster/whoopy)](LICENSE)

Unofficial WHOOP data CLI written in Go. `whoopy` wraps WHOOP’s OAuth flow and developer v2 APIs so humans, automations, dashboards, and AI agents can pull workouts, sleep, recovery, and stats securely from the terminal.

## Highlights

- 🏋️ **Full activity coverage** – profile, workouts (list/view/export), sleep, recovery, cycles, and day-level stats.
- 📊 **Daily dashboards** – `whoopy stats daily` aggregates workouts, recovery, sleep, and strain in one shot.
- 🛠 **Diagnostics built in** – `whoopy diag` surfaces config/tokens/API health for quick troubleshooting.
- 📦 **Deterministic outputs** – JSON by default for scripts, readable tables behind `--text`.
- 🔄 **HyperContext-ready export** – `whoopy hpx export` emits one canonical HyperContext NDJSON stream across profile, sleep, recovery, cycles, and workouts.
- 🔐 **First-party OAuth** – secure PKCE login, token persistence under `~/.config/whoopy`, automatic refresh, and one-click logout.
- 🧰 **Agent-friendly UX** – consistent flags, quiet success, non-zero exit codes on errors, and installable binaries for macOS/Linux arm64 + amd64.

## Installation

### Homebrew (recommended)

```bash
brew tap totocaster/tap
brew install whoopy
```

### Go install

```bash
go install github.com/toto/whoopy/cmd/whoopy@latest
```

### From source

```bash
git clone https://github.com/totocaster/whoopy.git
cd whoopy
make install   # builds and copies to ~/.local/bin/whoopy
```

`make build`, `make test`, and `make clean` are also available for local workflows.

## Configuration & Authentication

### 1. Create a WHOOP developer app

1. Sign in at [developer.whoop.com](https://developer.whoop.com/).
2. Navigate to **Apps → Create App** and fill in the metadata (name, description, etc.).
3. Add a **Redirect URI** – use `http://127.0.0.1:8735/oauth/callback` unless you have a reason to change it.
4. Copy the **Client ID** and **Client Secret** from the app detail screen; whoopy cannot authenticate without them.

> **Tip:** Keep the `offline` scope enabled plus the read scopes (`read:profile`, `read:workout`, etc.). whoopy requests the full set documented in [docs/initial_spec.md](docs/initial_spec.md#3-authentication--token-lifecycle).

### 2. Populate `config.toml`

Run `whoopy auth login` once. If `~/.config/whoopy/config.toml` (or `%APPDATA%\whoopy\config.toml` on Windows) does not exist, whoopy writes a template like:

```toml
client_id = "YOUR_CLIENT_ID"
client_secret = "YOUR_CLIENT_SECRET"

# Optional overrides; defaults shown for reference
api_base_url = "https://api.prod.whoop.com/developer/v2"
oauth_base_url = "https://api.prod.whoop.com/oauth"
redirect_uri = "http://127.0.0.1:8735/oauth/callback"
```

Update the two empty fields with your real credentials. You can also export environment variables (`WHOOPY_CLIENT_ID`, `WHOOPY_CLIENT_SECRET`, etc.) if you prefer not to store secrets on disk.

### 3. Complete the OAuth flow

Run `whoopy auth login` again after updating the config. By default:

- whoopy spins up a temporary HTTP listener on `127.0.0.1:8735` that matches the redirect URI.
- Your browser opens to WHOOP’s `/oauth/oauth2/auth` URL. Approve the requested scopes.
- The CLI receives the authorization code, exchanges it for access + refresh tokens, and writes them to `~/.config/whoopy/tokens.json`.

Headless workflows:

- `whoopy auth login --no-browser` prints the URL without trying to open it.
- `whoopy auth login --manual` or `--code "<redirect-url>"` lets you paste the callback URL instead of running a local listener.

Tokens refresh automatically before expiry (`whoopy auth status` shows the remaining lifetime). `whoopy auth logout` clears the local cache and attempts to revoke the refresh token.

### Config & token locations

| Platform | Config path | Tokens path |
| --- | --- | --- |
| macOS / Linux | `${XDG_CONFIG_HOME:-~/.config}/whoopy/config.toml` | `${XDG_CONFIG_HOME:-~/.config}/whoopy/tokens.json` |
| Windows | `%APPDATA%\whoopy\config.toml` | `%APPDATA%\whoopy\tokens.json` |

Override both via `WHOOPY_CONFIG_DIR` if you need custom locations (CI/CD, ephemeral environments, etc.).

Environment overrides:

| Variable | Purpose |
| --- | --- |
| `WHOOPY_CLIENT_ID`, `WHOOPY_CLIENT_SECRET` | Override config file credentials (e.g., in CI). |
| `WHOOPY_API_BASE_URL`, `WHOOPY_OAUTH_BASE_URL`, `WHOOPY_REDIRECT_URI` | Point at staging stacks or custom redirect ports. |
| `WHOOPY_CONFIG_DIR` | Custom config directory (defaults to XDG `.config/whoopy`). |

## Usage Overview

whoopy defaults to JSON output. Append `--text` for aligned tables or `| jq` for structured piping. All list commands accept `--start`, `--end`, `--limit`, and `--cursor` via shared pagination flags, plus bounded-export aliases `--since`, `--until`, and `--last`. Dates accept either `YYYY-MM-DD` or RFC3339 timestamps.

### Core commands

| Command | Description |
| --- | --- |
| `whoopy auth login/status/logout` | PKCE login, show token expiry + scopes, or revoke local tokens. |
| `whoopy hpx export [window flags]` | Auto-paginate profile + WHOOP collections into one deduplicated HyperContext NDJSON stream. |
| `whoopy profile show [--text]` | Basic profile plus body measurements. |
| `whoopy workouts list [filters]` | List workouts from `/activity/workout`; client-side filters `--sport`, `--min-strain`, `--max-strain`. |
| `whoopy workouts view <id>` | Detailed metrics for a single workout. |
| `whoopy workouts today` | Convenience alias for the current calendar day. |
| `whoopy workouts export [--format jsonl|csv] [--output PATH|-]` | Auto-paginates through WHOOP workouts and streams JSON Lines or CSV. |
| `whoopy sleep list/view/today` | Access `activity/sleep` collections. |
| `whoopy recovery list/view/today` | Fetch recovery scores and state. |
| `whoopy cycles list/view` | Pull daily strain cycles. |
| `whoopy stats daily --date YYYY-MM-DD [--text]` | Aggregate cycle, sleep, recovery, and workouts for a single day. |
| `whoopy diag [--text]` | Print config path, token status, and API health probe results. |
| `whoopy version` / `whoopy --version` | Embed build version, commit, and date (populated via GoReleaser ldflags). |

### Examples

```bash
# Quick sanity check
whoopy diag --text

# Workouts as JSON
whoopy workouts list --start 2026-03-01 --end 2026-03-04 | jq '.workouts[].score.strain'

# Human-readable sleep summary
whoopy sleep today --text

# HyperContext import for the last 10 days across all supported WHOOP resources
whoopy hpx export --last 10d | hpx import

# Filter workouts before exporting
whoopy workouts export \
  --start 2026-02-01 --end 2026-02-29 \
  --sport running \
  --min-strain 8 \
  --format csv \
  --output feb_running.csv

# Daily dashboard
whoopy stats daily --date 2026-03-03 --text
```

## HyperContext Export

`whoopy hpx export` is the HyperContext path for batch imports:

- It fetches profile, sleep, cycles, recovery, and workouts in one run
- It auto-paginates each WHOOP collection while applying one shared `--since` / `--until` / `--last` window
- It deduplicates overlapping `recovery_window` signposts that would otherwise appear when combining cycle and recovery exports
- Metric records now populate `ts` with the underlying measurement timestamp when WHOOP provides one
- Output contract: one JSON object per line on stdout, no tables or banners, `source` fixed to `whoop`, deterministic `origin_id`
- Session exports emit signposts before metrics so the stream can pipe directly into `hpx import`
- Current mappings cover registered HyperContext keys such as `sleep.duration_ms`, `recovery.score_pct`, `strain.day_score`, `workout.kilojoules`, and `body.weight_kg`

Bounded export guidance:

- Use `--since` / `--until` or the existing `--start` / `--end` flags for explicit windows
- Use `--last 10d` for overlap-friendly recurring imports
- `--updated-since` is intentionally rejected because WHOOP’s public endpoints do not expose a reliable updated-time filter

## Diagnostics

`whoopy diag` consolidates:

- **Config** – absolute path, existence, last modified timestamp, whether client credentials are set, and the effective API/OAuth/redirect URLs.
- **Tokens** – token file path, scopes, expiry, refresh-token presence, and friendly time-to-expiration.
- **API** – health check latency hitting `/user/profile/basic`, plus error messaging when auth/config is incomplete.

Use this command before filing bugs or when running on new machines to confirm tokens are valid.

## Development

```bash
# Format + test
gofmt -w .
go test ./...

# Build + install locally
make install

# Run CLI
whoopy --help
```

See [docs/initial_spec.md](docs/initial_spec.md) for deeper architectural notes and output contracts, plus [docs/AGENT.md](docs/AGENT.md) for contributor rituals (testing, documentation, release flow).

## License

MIT © 2026 [Toto Tvalavadze](https://ttvl.co)
