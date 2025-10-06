# Nebula Ops Deck Spec

## Summary
Create a Bubble Tea TUI called **Nebula Ops Deck** (`pzapp`) that lists OS ports, lets the user terminate processes with style, and blends a calm TokyoNight-inspired palette with a playful "port slayer" vibe. Keep provider abstractions intact while delivering animated accents, a rotating tag line, and emoji-rich feedback.

## Goals
- Present active ports in a compact, single-line table view with adaptive column widths.
- Deliver a dynamic header and footer with animated color accents and a rotating tag line.
- Provide a confident kill flow with expressive emoji copy, a confirmation modal, and clear status/toast messaging.
- Maintain the existing provider abstraction (real + mock) and Bubble Tea list experience.
- Keep runtime dependencies minimal (Bubble Tea, Bubbles list, Lip Gloss).

## Non-Goals
- No changes to port enumeration or termination logic beyond UI triggers.
- No data persistence or additional views (graphs, history, etc.).
- No new external services or tooling outside the current Go + Charmbracelet stack.

## UI Layout

### Screen Structure (Top → Bottom)
1. **Header** (three text lines plus a rule):
   - Title: `🌌 PZAPP`, tinted with the current primary accent color.
   - Tag line: animated cycle (see *Animations*).
   - Info line: `✨ Nebula ops deck · {count} active ports`, tinted with the tertiary accent.
   - Horizontal rule: full-width line rendered in the tertiary accent.
2. **Table Header**: column labels `PROTO | PORT | PROCESS | PID | USER | ADDRESS`, each styled per the accent pattern.
3. **Port List**: Bubble Tea list rows, each prefixed with `🛰`. Rows must stay single-line; truncate with ellipsis when necessary.
4. **Footer**: stacked lines for
   - Error (red) or status (muted accent),
   - Toast message (styled by toast kind),
   - Command hint line `💡 up/down or j/k move  |  enter/d kill  |  r refresh  |  / search  |  ? help  |  q quit`.
5. **Overlay States**: when active, dim the underlying header/list/table and show either
   - the kill confirmation modal, or
   - the help panel card.

### Kill Confirmation Modal
- Centered panel with width clamped between 30 and 72 columns.
- Subtitle: `💀🗡️ {process} on {PROTO}/{port}`.
- Status line: `💀 Ready to slay this port?` or `💀🌀 Dispatching SIGTERM...` when kill is in-flight.
- Buttons: primary `💀🗡️ [y] confirm`, secondary `[n] retreat`.

### Help Panel
- Centered card listing key bindings, matching the hints in the footer.

## Color System
- Base palette constants (hex):
  - `tokyoBase` `#1a1b26`
  - `tokyoSurface` `#1f2335`
  - `tokyoSurfaceAlt` `#24283b`
  - `tokyoSurfaceLine` `#2e3247`
  - `tokyoText` `#c0caf5`
  - `tokyoMutedText` `#a9b1d6`
  - `tokyoSubtleText` `#565f89`
  - Accents: `tokyoAccentBlue` `#7aa2f7`, `tokyoAccentPurple` `#bb9af7`, `tokyoAccentCyan` `#7dcfff`, `tokyoAccentGreen` `#9ece6a`, `tokyoAccentYellow` `#e0af68`, `tokyoAccentRed` `#f7768e`
- Accent cycle: `[tokyoAccentBlue, tokyoAccentPurple, tokyoAccentCyan, tokyoAccentGreen]`.
- Table header accent pattern: `[accent0, accent1, accent2, tokyoAccentYellow, tokyoAccentGreen, accent0]`.

## Animations
- Use `tea.Tick(120 * time.Millisecond)` to emit `tickMsg`.
- Maintain `tickCount`, `accentIndex`, and `taglineIndex` on the model.
- Every tick:
  - Increment `tickCount`.
  - Every 3 ticks: advance `accentIndex` (modulo accent cycle) and reapply accent styles.
  - Every 8 ticks: advance `taglineIndex` (modulo tagline cycle).
  - Clear toast state when past expiry time.
- Tag line cycle strings:
  1. `⚡ zap those ports ⚡`
  2. `🗡️ slay stray sockets 🗡️`
  3. `🚀 keep your stack lean 🚀`

## Copy & Emoji
- Loading status: `🔍 Loading active ports...`
- Load success: `✨ Loaded {n} ports @ {time}`
- Refresh (`r`): status `🔄 Refreshing...`
- Kill target selected: status `💀🗡️ Target locked: {process} ({pid})`
- Kill queued: toast `💀🗡️ Priming SIGTERM for PID {pid}...`
- Kill success: toast `✅ Terminated {process} ({pid})`, status `🔄 Refreshing port list...`
- Kill failure: toast `⚠️ Failed to terminate {process} ({pid})`, error `termination failed: {err}`
- Hint line: `💡 up/down or j/k move  |  enter/d kill  |  r refresh  |  / search  |  ? help  |  q quit`

## User Interactions
- Navigation: Up/Down arrows or `j/k`.
- Filter: `/` enters filter mode (no placeholder glyphs); `esc` exits.
- Toggle help: `?`.
- Refresh: `r`.
- Quit: `q` or `ctrl+c`.
- Kill flow:
  - `enter` or `d` opens confirmation modal.
  - While modal is open:
    - `y` or `enter` executes kill (respecting `killPending` guard).
    - `n` or `esc` cancels.
- When overlays (modal/help) are visible, dim the main view and pause list navigation.

## Data & Providers
- Reuse the provider interface: `ports.Provider` with `List(ctx)` and `Terminate(pid)`.
- Application must work with both the real provider (lsof) and the mock provider (`PZAPP_USE_MOCK=1`).
- Load ports on startup, after successful kills, and on manual refresh.

## Rendering Constraints
- Maintain adaptive `columnWidths` logic to keep rows single-line.
- Use `padded` helper to truncate with ellipsis when strings exceed column widths.
- Keep all base text ASCII except specified emoji.
- Dim header/table/list when overlays are active.

## Model State
- Fields must include: provider, list model, status message, error message, toast state, dimensions, ready flag, confirmation pointer (`*ports.Port`), `killPending`, column widths, `helpVisible`, `accentIndex`, `taglineIndex`, `tickCount`.
- Helpers: `applyAccentStyles()`, `accentColor(offset)`, `resizeList()`, `recalcColumns()`, `removeEntry()`.

## Testing & Validation
- Automated: `go test ./...` must pass.
- Manual demo checklist:
  1. `PZAPP_USE_MOCK=1 go run ./cmd/pzapp`
  2. Observe accent pulse (every ~360ms) and slower tag line rotation (~960ms).
  3. Select a port; confirm status/toast/modal copy uses `💀🗡️`.
  4. Run through confirm and cancel paths.
  5. Simulate failure (via mock) to verify error messaging.
  6. Exercise filter, help toggle, refresh, and quit.

## Project Setup
- Repo layout:
  - `cmd/pzapp/main.go` – entry point configuring providers and running Bubble Tea program.
  - `internal/ui` – Bubble Tea model, styles, animations, modals.
  - `internal/ports` – provider interface, implementations, tests.
- Tests colocated with implementation files.
- Include this `SPEC.md` at repo root for reference.

