# Repository Guidelines

## Project Structure & Module Organization
- `cmd/pzapp`: CLI entry point configuring providers and launching the Bubble Tea program.
- `internal/ui`: Bubble Tea model, shared styles, and animation logic for the pzapp TUI.
- `internal/ports`: Providers for enumerating ports (`LsofProvider`, `MockProvider`) and helpers such as parser tests and Unix termination.
- Tests live beside the code they cover (e.g., `internal/ports/lsof_test.go`). Keep new packages under `internal/` unless they’re shared externally.

## Build, Test, and Development Commands
- `go run ./cmd/pzapp`: Start the real pzapp TUI (uses `lsof`; requires Unix permissions to kill PIDs).
- `PZAPP_USE_MOCK=1 go run ./cmd/pzapp`: Launch with deterministic mock data for UI tweaks or demos.
- `go test ./...`: Execute all package tests, including parser coverage for the `ports` package.
- `gofmt -w <files>`: Apply canonical Go formatting to modified files before committing.

## Coding Style & Naming Conventions
- Follow Go defaults: tabs for indentation, mixedCaps for exported identifiers, and short, contextual receiver names.
- Keep UI strings ASCII and prefer concise gradients defined near other Lip Gloss styles.
- Group related helpers or styles in the same file; avoid cross-package globals outside `internal/ui`.
- Always format Go files with `gofmt`; optional `goimports` is welcome for import tidiness.

## Testing Guidelines
- Use Go’s standard `testing` package; continue storing tests next to implementation files.
- Name tests `Test<Subject>` and cover parsing edge cases (duplicate rows, IPv6) when modifying `ports`.
- When adding new providers, include integration-friendly tests guarded by build tags or mocks to avoid network/process dependencies.

## Commit & Pull Request Guidelines
- Commits should be scoped and descriptive (`ui: animate gradient tick`, `ports: harden lsof parser`). Favor the present tense.
- Pull requests should summarize behavior changes, list testing commands run, and attach screenshots/gifs for UI updates (especially TUI styling tweaks).
- Reference Linear or issue IDs when applicable, and call out any permissions required for `lsof` or process signals.
