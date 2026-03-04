# whoopy

[![Release](https://img.shields.io/github/v/release/totocaster/whoopy)](https://github.com/totocaster/whoopy/releases)
[![Build](https://github.com/totocaster/whoopy/actions/workflows/release.yml/badge.svg)](https://github.com/totocaster/whoopy/actions/workflows/release.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/totocaster/whoopy)](https://go.dev/)
[![License](https://img.shields.io/github/license/totocaster/whoopy)](LICENSE)

Official WHOOP data CLI written in Go. whoopy wraps WHOOP’s OAuth flow and developer v2 APIs so automations, dashboards, and AI agents can pull workouts, sleep, recovery, and stats securely from the terminal.

## Highlights

- 🔐 **First-party OAuth** – secure PKCE login, token persistence under `~/.config/whoopy`, automatic refresh, and one-click logout.
- 📦 **Deterministic outputs** – JSON by default for scripts, readable tables behind `--text`, and stable schemas documented in `docs/initial_spec.md`.
- 🏋️ **Full activity coverage** – profile, workouts (list/view/export), sleep, recovery, cycles, and day-level stats.
- 📊 **Daily dashboards** – `whoopy stats daily` aggregates workouts, recovery, sleep, and strain in one shot.
- 🛠 **Diagnostics built in** – `whoopy diag` surfaces config/tokens/API health for quick troubleshooting.
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

1. Create a WHOOP developer app and note the **Client ID** and **Client Secret**.
2. Run `whoopy auth login`. The CLI will scaffold `~/.config/whoopy/config.toml` like:

```toml
client_id = "your-client-id"
client_secret = "your-client-secret"
oauth_base_url = "https://api.prod.whoop.com/oauth"
redirect_uri = "http://127.0.0.1:8735/oauth/callback"
```

3. The login flow defaults to opening your browser and listening on `127.0.0.1:8735`. Use `--manual` or `--no-browser` when working on remote machines.
4. Tokens are stored at `~/.config/whoopy/tokens.json` (`%APPDATA%\whoopy\tokens.json` on Windows) with `0600` permissions. whoopy refreshes tokens automatically and rotates them when WHOOP returns new values.

Environment overrides:

| Variable | Purpose |
| --- | --- |
| `WHOOPY_CLIENT_ID`, `WHOOPY_CLIENT_SECRET` | Override config file credentials (e.g., in CI). |
| `WHOOPY_API_BASE_URL`, `WHOOPY_OAUTH_BASE_URL`, `WHOOPY_REDIRECT_URI` | Point at staging stacks or custom redirect ports. |
| `WHOOPY_CONFIG_DIR` | Custom config directory (defaults to XDG `.config/whoopy`). |

## Usage Overview

whoopy defaults to JSON output. Append `--text` for aligned tables or `| jq` for structured piping. All list commands accept `--start`, `--end`, `--limit`, and `--cursor` via shared pagination flags. Dates accept either `YYYY-MM-DD` or RFC3339 timestamps.

### Core commands

| Command | Description |
| --- | --- |
| `whoopy auth login/status/logout` | PKCE login, show token expiry + scopes, or revoke local tokens. |
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

## Diagnostics

`whoopy diag` consolidates:

- **Config** – absolute path, existence, last modified timestamp, whether client credentials are set, and the effective API/OAuth/redirect URLs.
- **Tokens** – token file path, scopes, expiry, refresh-token presence, and friendly time-to-expiration.
- **API** – health check latency hitting `/user/profile/basic`, plus error messaging when auth/config is incomplete.

Use this command before filing bugs or when running on new machines to confirm tokens are valid.

## Releases & Versioning

- Semantic versioning with tags `vMAJOR.MINOR.PATCH`.
- GoReleaser builds macOS/Linux binaries (arm64 + amd64), publishes GitHub Releases, and updates the Homebrew tap (`totocaster/homebrew-tap`).
- Release process is documented in [RELEASE_SETUP.md](RELEASE_SETUP.md). In short: tag `vX.Y.Z`, push, watch the `Release` workflow, then `brew install whoopy` to smoke test.

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

MIT © [toto](https://github.com/totocaster)
