# Whoopy Agent Notes

## Routine & Rituals
1. **Spec Alignment** – Before coding, review `docs/initial_spec.md` and update it with new decisions or status changes.
2. **Implementation** – Write minimal, well-structured code adhering to `clig.dev` guidelines (quiet success, clear errors, subcommands, helpful `--help` examples).
3. **Testing Requirement** – Every feature must ship with automated coverage. Add unit/integration tests touching new code paths; use `go test ./...`. Only commit after tests pass.
4. **Documentation** – Update `docs/initial_spec.md`, this `AGENT.md`, and any CLI help/docs so future agents understand what changed.
5. **Commit Discipline** – Keep commits small and logical (one feature/change per commit). Never commit without green tests.

## Current Status (2026-03-04)
- Repo bootstrapped with Go module, Cobra root command, config loader, and token store.
- OAuth PKCE flow implemented with `whoopy auth login|status|logout`.
- Next priority: WHOOP API client + data commands (profile, cycles, recovery, sleep, workouts, stats).

## Testing Checklist
- `go test ./...` before every commit.
- Future work: add targeted tests for config loading, token storage, PKCE helpers, and CLI logic using e.g. `testing` + `httptest`.
