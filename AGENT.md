# Whoopy Agent Notes

## Routine & Rituals
1. **Spec Alignment** – Before coding, review `docs/initial_spec.md` and update it with new decisions or status changes.
2. **Implementation** – Write minimal, well-structured code adhering to `clig.dev` guidelines (quiet success, clear errors, subcommands, helpful `--help` examples).
3. **Testing Requirement** – Every feature must ship with automated coverage. Add unit/integration tests touching new code paths; use `go test ./...`. Only commit after tests pass.
4. **Binary Verification** – After `go test ./...`, run `make install` (which builds + copies to `~/.local/bin/whoopy`) and execute at least one representative CLI command (e.g., `whoopy <feature> ...`) to ensure the installed binary works end-to-end.
5. **Documentation** – Update `docs/initial_spec.md`, this `AGENT.md`, and any CLI help/docs so future agents understand what changed.
6. **Commit Discipline** – Keep commits small and logical (one feature/change per commit). Never commit without green tests.
7. **Release Ritual** – When publishing, follow `RELEASE_SETUP.md`: tag `vX.Y.Z`, push, watch the GoReleaser workflow, and verify the Homebrew tap update + `brew install whoopy`.

## Current Status (2026-03-07)
- Auth stack complete (`whoopy auth login|status|logout`) with persisted tokens + auto-refresh.
- Core API client + shared list plumbing ready; profile summary implemented.
- Workouts service + CLI (`whoopy workouts list/view`, JSON + `--text`) now include client-side filters (`--sport`, `--min-strain`, `--max-strain`) while hitting `/developer/v2/activity/workout`. New `whoopy workouts today` and `whoopy workouts export --format jsonl|csv` convenience commands reuse the same filtering and auto-pagination helper.
- Cycles, recovery, and sleep services/CLI pairs (`whoopy cycles|recovery|sleep list/view`) implemented with shared pagination + formatting, plus `today` helpers for quick snapshots.
- Recovery and sleep each gained `today` shortcuts mirroring the workouts UX for quick daily snapshots.
- Stats aggregation landed (`whoopy stats daily --date …`) producing JSON/text dashboards by composing cycles/recovery/sleep/workouts.
- Diagnostics command (`whoopy diag`) now surfaces config/token paths, credential presence, token expiry, and API probe status.
- Shared bounded range aliases `--since`, `--until`, and `--last` are available on list-style commands. `--updated-since` remains intentionally unsupported because WHOOP does not expose a trustworthy updated-time filter.
- Release tooling added: `.goreleaser.yml`, Homebrew tap config, and Release workflow ready for v0.1.0 (see `RELEASE_SETUP.md`).
- Next up: any remaining distribution niceties (e.g., Windows builds).

## Testing Checklist
- `go test ./...` before every commit.
- Future work: add targeted tests for config loading, token storage, PKCE helpers, and CLI logic using e.g. `testing` + `httptest`.
