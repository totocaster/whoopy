# Whoopy Agent Notes

## Routine & Rituals
1. **Spec Alignment** – Before coding, review `docs/initial_spec.md` and update it with new decisions or status changes.
2. **Implementation** – Write minimal, well-structured code adhering to `clig.dev` guidelines (quiet success, clear errors, subcommands, helpful `--help` examples).
3. **Testing Requirement** – Every feature must ship with automated coverage. Add unit/integration tests touching new code paths; use `go test ./...`. Only commit after tests pass.
4. **Binary Verification** – After `go test ./...`, run `go install ./cmd/whoopy` and execute at least one representative CLI command (e.g., `whoopy <feature> ...`) to ensure the installed binary works end-to-end.
5. **Documentation** – Update `docs/initial_spec.md`, this `AGENT.md`, and any CLI help/docs so future agents understand what changed.
6. **Commit Discipline** – Keep commits small and logical (one feature/change per commit). Never commit without green tests.

## Current Status (2026-03-04)
- Auth stack complete (`whoopy auth login|status|logout`) with persisted tokens + auto-refresh.
- Core API client + shared list plumbing ready; profile summary implemented.
- Workouts service + CLI (`whoopy workouts list/view`, JSON + `--text`) now include client-side filters (`--sport`, `--min-strain`, `--max-strain`) while hitting `/developer/v2/activity/workout`.
- Cycles, recovery, and sleep services/CLI pairs (`whoopy cycles|recovery|sleep list/view`) implemented with shared pagination + formatting.
- Stats aggregation landed (`whoopy stats daily --date …`) producing JSON/text dashboards by composing cycles/recovery/sleep/workouts.
- Next up: diagnostics/export tooling and polish around release packaging.

## Testing Checklist
- `go test ./...` before every commit.
- Future work: add targeted tests for config loading, token storage, PKCE helpers, and CLI logic using e.g. `testing` + `httptest`.
