# Session Handoff

## Completed Work
- Added labeled column header row and dimming integration with modal/help overlays (`internal/ui/model.go`).
- Styled and tightened filter/title bar to eliminate the stray square placeholder when filtering is inactive.
- Reworked column layout to prevent wrapping: compact widths, consistent separators, PID column, and inline process state for single-line rows (same file).
- Disabled list descriptions and refreshed toast/status handling to fit the new layout.
- Verified `go test ./...` after each change; UI validated manually with mock provider.

## Release Process Proposal
- Use GitHub Actions workflow triggered on `v*.*.*` tags to run tests and execute `goreleaser release --clean` on `ubuntu-latest`.
- Configure `.goreleaser.yaml` to build binaries for macOS (amd64/arm64) and Linux (amd64/arm64), generate checksums, and upload tarballs plus raw binaries.
- Leverage GoReleaser `brews` section to update a Homebrew tap (`pzapp/homebrew-tap`) so users install via `brew install pzapp/tap/pzapp`.
- Provide a curl/tar install snippet in README for Linux users who skip Homebrew; optionally add `.deb` packaging later via GoReleaser `nfpms`.
- Internal flow: maintain `CHANGELOG.md`, run `goreleaser check`, tag with `git tag -s vX.Y.Z`, push tag, confirm release assets and tap update post-pipeline.
- Add Dependabot entry for GoReleaser and a periodic dry-run workflow (`goreleaser release --skip-publish --snapshot`) to catch config drift.

## Next Session Targets
- Commit the GoReleaser config and GitHub Actions workflow.
- Create/initialize the Homebrew tap repo and test tap update automation.
- Document installation methods in README (brew + curl snippet) once the first release is ready.
