package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"portkiller/internal/ports"

	list "github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Model implements the Bubble Tea program for pzapp.
type Model struct {
	provider ports.Provider

	list      list.Model
	statusMsg string
	errMsg    string
	width     int
	height    int
	ready     bool

	confirm     *ports.Port
	killPending bool

	toast        toastState
	columns      columnWidths
	helpVisible  bool
	accentIndex  int
	taglineIndex int
	tickCount    int
}

type portsLoadedMsg struct {
	entries []ports.Port
	err     error
}

type killResultMsg struct {
	entry ports.Port
	err   error
}

type tickMsg struct {
	when time.Time
}

type toastKind int

const (
	toastNone toastKind = iota
	toastInfo
	toastSuccess
	toastError
)

type toastState struct {
	message string
	kind    toastKind
	expires time.Time
}

type columnWidths struct {
	proto   int
	port    int
	process int
	pid     int
	user    int
	address int
}

const columnSeparator = " | "

func defaultColumns() columnWidths {
	return columnWidths{
		proto:   4,
		port:    5,
		process: 16,
		pid:     6,
		user:    9,
		address: 28,
	}
}

// New creates the root Bubble Tea model.
func New(provider ports.Provider) Model {
	baseDelegate := list.NewDefaultDelegate()
	baseDelegate.ShowDescription = false
	baseDelegate.SetSpacing(0)
	baseDelegate.Styles.NormalTitle = listTitleStyle
	baseDelegate.Styles.SelectedTitle = selectedTitleBase.Background(lipgloss.Color(tokyoAccentBlue))
	baseDelegate.Styles.NormalDesc = listDescStyle
	baseDelegate.Styles.SelectedDesc = selectedDescBase.Background(lipgloss.Color(tokyoAccentPurple))

	model := Model{provider: provider}

	l := list.New([]list.Item{}, baseDelegate, 0, 0)
	l.Title = ""
	l.SetShowTitle(false)
	l.SetShowPagination(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.Styles.HelpStyle = helpStyle
	l.Styles.FilterCursor = filterCursorBase.Foreground(lipgloss.Color(tokyoAccentBlue))
	l.Styles.FilterPrompt = filterPromptStyle
	l.Styles.TitleBar = filterBarStyle

	// Update the pre-created model with list
	model.list = l
	model.statusMsg = "🔍 Loading active ports..."
	model.columns = defaultColumns()
	model.applyAccentStyles()

	return model
}

// Init starts the asynchronous refresh when the program boots.
func (m Model) Init() tea.Cmd {
	return tea.Batch(tea.EnterAltScreen, loadPortsCmd(m.provider), animationTickCmd())
}

// Update applies incoming Bubble Tea messages to the model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.resizeList()
		return m, nil

	case portsLoadedMsg:
		if msg.err != nil {
			m.errMsg = fmt.Sprintf("error loading ports: %v", msg.err)
			m.statusMsg = ""
			return m, nil
		}

		items := make([]list.Item, 0, len(msg.entries))
		for _, entry := range msg.entries {
			items = append(items, portItem{entry: entry, layout: &m.columns})
		}
		m.list.SetItems(items)
		m.errMsg = ""
		m.recalcColumns()
		m.statusMsg = fmt.Sprintf("✨ Loaded %d ports @ %s", len(items), time.Now().Format(time.Kitchen))
		return m, nil

	case killResultMsg:
		m.killPending = false
		m.confirm = nil
		m.resizeList()
		if msg.err != nil {
			m.toast = newToast(fmt.Sprintf("⚠️ Failed to terminate %s (%d)", msg.entry.Process, msg.entry.PID), toastError)
			m.errMsg = fmt.Sprintf("termination failed: %v", msg.err)
		} else {
			m.removeEntry(msg.entry)
			m.toast = newToast(fmt.Sprintf("✅ Terminated %s (%d)", msg.entry.Process, msg.entry.PID), toastSuccess)
			m.statusMsg = "🔄 Refreshing port list..."
			cmds = append(cmds, loadPortsCmd(m.provider))
		}
		return m, tea.Batch(cmds...)

	case tickMsg:
		m.tickCount++
		if len(accentCycle) > 0 && m.tickCount%3 == 0 {
			m.accentIndex = (m.accentIndex + 1) % len(accentCycle)
			m.applyAccentStyles()
		}
		if len(taglineCycle) > 0 && m.tickCount%8 == 0 {
			m.taglineIndex = (m.taglineIndex + 1) % len(taglineCycle)
		}
		if m.toast.message != "" && msg.when.After(m.toast.expires) {
			m.toast = toastState{}
		}
		return m, animationTickCmd()

	case tea.KeyMsg:
		if m.confirm != nil {
			switch msg.String() {
			case "y", "Y", "enter":
				if !m.killPending {
					entry := *m.confirm
					m.killPending = true
					m.toast = newToast(fmt.Sprintf("💀🗡️ Priming SIGTERM for PID %d...", entry.PID), toastInfo)
					cmds = append(cmds, killProcessCmd(entry))
				}
			case "n", "N", "esc":
				m.confirm = nil
				m.killPending = false
				m.resizeList()
			}
			if len(cmds) > 0 {
				return m, tea.Batch(cmds...)
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.helpVisible {
				m.helpVisible = false
				m.resizeList()
			}
		case "r":
			m.statusMsg = "🔄 Refreshing..."
			cmds = append(cmds, loadPortsCmd(m.provider))
		case "/":
			// fall through to list for filtering shortcut.
		case "?":
			m.helpVisible = !m.helpVisible
			m.resizeList()
			return m, nil
		case "enter", "d":
			if item, ok := m.list.SelectedItem().(portItem); ok {
				entry := item.entry
				m.confirm = &entry
				m.killPending = false
				m.statusMsg = fmt.Sprintf("💀🗡️ Target locked: %s (%d)", entry.Process, entry.PID)
				m.resizeList()
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	if m.confirm == nil && !m.helpVisible {
		m.list, cmd = m.list.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	if len(cmds) == 0 {
		return m, nil
	}
	return m, tea.Batch(cmds...)
}

// View renders the current UI to a string.
func (m Model) View() string {
	if !m.ready {
		return "\n  Booting pzapp UI..."
	}

	tableHeader := m.renderTableHeader()
	listView := m.list.View()
	if m.confirm != nil || m.helpVisible {
		if tableHeader != "" {
			tableHeader = dimStyle.Render(tableHeader)
		}
		listView = dimStyle.Render(listView)
	}

	sections := []string{m.renderHeader()}
	if tableHeader != "" {
		sections = append(sections, tableHeader)
	}
	sections = append(sections,
		listView,
		m.renderFooter(),
	)

	var modal string

	if m.helpVisible {
		sections = append(sections, "", renderHelp(m.width))
	} else if m.confirm != nil {
		modal = renderKillModal(*m.confirm, m.killPending, m.width)
	}

	view := strings.Join(sections, "\n")
	if modal != "" {
		return overlayModal(view, modal, m.width, m.height)
	}

	return view
}

func (m *Model) resizeList() {
	if m.width == 0 || m.height == 0 {
		return
	}

	reserve := headerLines + tableHeaderLines + footerLines
	switch {
	case m.helpVisible:
		reserve += helpLines
	}

	height := max(3, m.height-reserve)
	m.list.SetSize(m.width, height)
	m.recalcColumns()
}

func (m *Model) recalcColumns() {
	width := m.list.Width()
	if width <= 0 {
		return
	}

	separatorWidth := lipgloss.Width(columnSeparator)
	const separatorCount = 5

	available := width - separatorWidth*separatorCount
	if available < 5 {
		available = 5
	}

	columns := columnWidths{}
	specs := []struct {
		field   *int
		min     int
		desired int
	}{
		{&columns.proto, 3, 4},
		{&columns.port, 4, 5},
		{&columns.process, 12, 18},
		{&columns.pid, 5, 6},
		{&columns.user, 6, 12},
		{&columns.address, 16, 30},
	}

	remaining := available
	for i := range specs {
		*specs[i].field = specs[i].min
		remaining -= specs[i].min
	}

	if remaining < 0 {
		deficit := -remaining
		order := []int{len(specs) - 1, 3, 4, 1, 0, 2}
		for _, idx := range order {
			if deficit == 0 {
				break
			}
			current := *specs[idx].field
			reducible := current - 1
			if reducible <= 0 {
				continue
			}
			cut := min(deficit, reducible)
			*specs[idx].field = current - cut
			deficit -= cut
		}
		remaining = 0
	}

	for remaining > 0 {
		progress := false
		for i := range specs {
			if remaining == 0 {
				break
			}
			current := *specs[i].field
			if current >= specs[i].desired {
				continue
			}
			add := min(specs[i].desired-current, remaining)
			*specs[i].field = current + add
			remaining -= add
			progress = true
		}
		if !progress {
			for i := range specs {
				if remaining == 0 {
					break
				}
				*specs[i].field++
				remaining--
			}
		}
	}

	m.columns = columns
}

func (m *Model) removeEntry(entry ports.Port) {
	items := m.list.Items()
	if len(items) == 0 {
		return
	}

	filtered := make([]list.Item, 0, len(items)-1)
	removed := false
	for _, item := range items {
		pi, ok := item.(portItem)
		if !ok {
			filtered = append(filtered, item)
			continue
		}
		if !removed && pi.entry.PID == entry.PID && pi.entry.Port == entry.Port && strings.EqualFold(pi.entry.Protocol, entry.Protocol) {
			removed = true
			continue
		}
		pi.layout = &m.columns
		filtered = append(filtered, pi)
	}

	if !removed {
		return
	}

	m.list.SetItems(filtered)
	if len(filtered) > 0 {
		idx := m.list.Index()
		if idx >= len(filtered) {
			m.list.Select(len(filtered) - 1)
		}
	}
	m.recalcColumns()
}

func loadPortsCmd(p ports.Provider) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		entries, err := p.List(ctx)
		return portsLoadedMsg{entries: entries, err: err}
	}
}

func killProcessCmd(entry ports.Port) tea.Cmd {
	return func() tea.Msg {
		err := ports.Terminate(entry.PID)
		return killResultMsg{entry: entry, err: err}
	}
}

func animationTickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg{when: t}
	})
}

type portItem struct {
	entry  ports.Port
	layout *columnWidths
}

func (p portItem) Title() string {
	layout := defaultColumns()
	if p.layout != nil {
		layout = *p.layout
	}
	proto := strings.ToUpper(p.entry.Protocol)
	state := strings.ToLower(p.entry.State)
	if state == "" {
		state = "listening"
	}
	process := p.entry.Process
	if state != "" {
		process = fmt.Sprintf("%s (%s)", process, state)
	}

	columns := []string{
		padded(proto, layout.proto),
		padded(fmt.Sprintf("%d", p.entry.Port), layout.port),
		padded(process, layout.process),
		padded(fmt.Sprintf("%d", p.entry.PID), layout.pid),
		padded(p.entry.User, layout.user),
		padded(p.entry.Address, layout.address),
	}
	return "🛰  " + strings.Join(columns, columnSeparator)
}

func (p portItem) Description() string {
	return ""
}

func (p portItem) FilterValue() string {
	return fmt.Sprintf("%s %d %s %s %s", p.entry.Process, p.entry.Port, p.entry.Protocol, p.entry.User, p.entry.State)
}

func newToast(message string, kind toastKind) toastState {
	return toastState{message: message, kind: kind, expires: time.Now().Add(3 * time.Second)}
}

func padded(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(text) > width {
		runes := []rune(text)
		limit := min(len(runes), max(1, width-1))
		return string(runes[:limit]) + "..."
	}
	return text + strings.Repeat(" ", max(0, width-lipgloss.Width(text)))
}

func (m Model) accentColor(offset int) string {
	if len(accentCycle) == 0 {
		return tokyoAccentBlue
	}
	idx := (m.accentIndex + offset) % len(accentCycle)
	if idx < 0 {
		idx += len(accentCycle)
	}
	return accentCycle[idx]
}

func (m *Model) applyAccentStyles() {
	accentMain := lipgloss.Color(m.accentColor(0))
	accentSoft := lipgloss.Color(m.accentColor(2))

	m.list.Styles.FilterCursor = filterCursorBase.Foreground(accentMain)
	m.list.Styles.TitleBar = filterBarStyle.Copy().BorderForeground(accentSoft)
}

func (m Model) renderHeader() string {
	if m.width == 0 {
		return ""
	}

	accentPrimary := lipgloss.Color(m.accentColor(0))
	accentSecondary := lipgloss.Color(m.accentColor(1))
	accentTertiary := lipgloss.Color(m.accentColor(2))

	title := headerTitleBase.Foreground(accentPrimary).Render("🌌 PZAPP")
	taglineText := "⚡ zap those ports ⚡"
	if len(taglineCycle) > 0 {
		taglineText = taglineCycle[m.taglineIndex%len(taglineCycle)]
	}
	tagline := headerTaglineBase.Foreground(accentSecondary).Render(taglineText)
	infoText := fmt.Sprintf("✨ Nebula ops deck · %d active ports", len(m.list.Items()))
	info := headerSubtitleStyle.Foreground(accentTertiary).Render(infoText)
	border := headerBorderBase.Foreground(accentTertiary).Render(strings.Repeat("─", max(0, m.width)))

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.PlaceHorizontal(m.width, lipgloss.Left, title),
		lipgloss.PlaceHorizontal(m.width, lipgloss.Left, tagline),
		lipgloss.PlaceHorizontal(m.width, lipgloss.Left, info),
		border,
	)
}

func (m Model) renderTableHeader() string {
	if m.width == 0 {
		return ""
	}

	columnTexts := []string{"PROTO", "PORT", "PROCESS", "PID", "USER", "ADDRESS"}
	columnWidths := []int{m.columns.proto, m.columns.port, m.columns.process, m.columns.pid, m.columns.user, m.columns.address}
	accentSequence := []string{
		m.accentColor(0),
		m.accentColor(1),
		m.accentColor(2),
		tokyoAccentYellow,
		tokyoAccentGreen,
		m.accentColor(0),
	}
	styledColumns := make([]string, len(columnTexts))
	for i, text := range columnTexts {
		style := tableHeaderBase.Foreground(lipgloss.Color(accentSequence[i%len(accentSequence)]))
		styledColumns[i] = style.Render(padded(text, columnWidths[i]))
	}

	row := strings.Join(styledColumns, tableSeparatorStyle.Render(columnSeparator))
	return lipgloss.PlaceHorizontal(m.width, lipgloss.Left, row)
}

func (m Model) renderFooter() string {
	var lines []string
	if m.errMsg != "" {
		lines = append(lines, errorStyle.Render(m.errMsg))
	} else if m.statusMsg != "" {
		lines = append(lines, statusStyle.Render(m.statusMsg))
	}

	if m.toast.message != "" {
		lines = append(lines, toastStyle(m.toast))
	}

	hint := hintStyle.Render("💡 up/down or j/k move  |  enter/d kill  |  r refresh  |  / search  |  ? help  |  q quit")
	lines = append(lines, hint)

	rendered := make([]string, len(lines))
	for i, line := range lines {
		rendered[i] = lipgloss.PlaceHorizontal(m.width, lipgloss.Left, line)
	}
	return strings.Join(rendered, "\n")
}

func renderKillModal(entry ports.Port, inFlight bool, width int) string {
	subtitle := fmt.Sprintf("💀🗡️ %s on %s/%d", entry.Process, strings.ToUpper(entry.Protocol), entry.Port)
	status := "💀 Ready to slay this port?"
	if inFlight {
		status = "💀🌀 Dispatching SIGTERM..."
	}

	modalWidth := clamp(width-4, 30, 72)
	innerWidth := modalWidth - modalStyle.GetPaddingLeft() - modalStyle.GetPaddingRight()
	if innerWidth < 16 {
		innerWidth = 16
	}

	lines := []string{
		modalTitleBase.Render("Confirm termination"),
		modalSubtitleStyle.Render(subtitle),
		"",
		modalStatusBase.Render(status),
		lipgloss.JoinHorizontal(lipgloss.Left,
			modalConfirmBase.Render("💀🗡️ [y] confirm"),
			modalActionSpacer.Render("  "),
			modalCancelBase.Render("[n] retreat"),
		),
	}

	contentLines := make([]string, len(lines))
	for i, line := range lines {
		contentLines[i] = modalContentStyle.Width(innerWidth).Render(line)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, contentLines...)
	return modalStyle.Width(modalWidth).Render(content)
}

func renderHelp(width int) string {
	items := []string{
		"navigate", "j/k", "arrow keys also work",
		"kill", "enter/d", "select then confirm",
		"refresh", "r", "reload port data",
		"search", "/", "filter by text",
		"help", "?", "toggle this panel",
	}

	rows := make([]string, 0, len(items)/3)
	for i := 0; i < len(items); i += 3 {
		rows = append(rows, fmt.Sprintf("%-10s %-10s %s", items[i], items[i+1], items[i+2]))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, rows...)
	panel := helpPanelStyle.Render(body)
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, panel)
}

func toastStyle(t toastState) string {
	switch t.kind {
	case toastSuccess:
		return successToastStyle.Render(t.message)
	case toastError:
		return errorToastStyle.Render(t.message)
	default:
		return infoToastStyle.Render(t.message)
	}
}

func overlayModal(base, modal string, width, height int) string {
	if width <= 0 {
		return base
	}

	baseLines := strings.Split(base, "\n")
	if height > 0 {
		switch {
		case len(baseLines) < height:
			pad := make([]string, height-len(baseLines))
			baseLines = append(baseLines, pad...)
		case len(baseLines) > height:
			baseLines = baseLines[:height]
		}
	}

	blockHeight := len(baseLines)
	if blockHeight == 0 {
		blockHeight = max(height, 0)
	}
	if blockHeight == 0 {
		return base
	}

	lineStyle := lipgloss.NewStyle().Width(width)
	for i := range baseLines {
		baseLines[i] = lineStyle.Render(baseLines[i])
	}

	modalBlock := lipgloss.Place(width, blockHeight, lipgloss.Center, lipgloss.Center, modal,
		lipgloss.WithWhitespaceChars(" "))
	modalLines := strings.Split(modalBlock, "\n")

	for i := 0; i < len(baseLines) && i < len(modalLines); i++ {
		if strings.TrimSpace(ansi.Strip(modalLines[i])) == "" {
			continue
		}
		baseLines[i] = modalLines[i]
	}

	return strings.Join(baseLines, "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clamp(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

const (
	tokyoBase         = "#1a1b26"
	tokyoSurface      = "#1f2335"
	tokyoSurfaceAlt   = "#24283b"
	tokyoSurfaceLine  = "#2e3247"
	tokyoText         = "#c0caf5"
	tokyoMutedText    = "#a9b1d6"
	tokyoSubtleText   = "#565f89"
	tokyoAccentBlue   = "#7aa2f7"
	tokyoAccentCyan   = "#7dcfff"
	tokyoAccentPurple = "#bb9af7"
	tokyoAccentGreen  = "#9ece6a"
	tokyoAccentYellow = "#e0af68"
	tokyoAccentRed    = "#f7768e"
)

var accentCycle = []string{
	tokyoAccentBlue,
	tokyoAccentPurple,
	tokyoAccentCyan,
	tokyoAccentGreen,
}

var taglineCycle = []string{
	"⚡ zap those ports ⚡",
	"🗡️ slay stray sockets 🗡️",
	"🚀 keep your stack lean 🚀",
}

var (
	listTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(tokyoText)).
			Background(lipgloss.Color(tokyoSurface)).
			PaddingLeft(1)

	listDescStyle = listTitleStyle.Copy().
			Foreground(lipgloss.Color(tokyoMutedText))

	selectedTitleBase = lipgloss.NewStyle().
				Foreground(lipgloss.Color(tokyoBase)).
				Bold(true).
				PaddingLeft(1)

	selectedDescBase = lipgloss.NewStyle().
				Foreground(lipgloss.Color(tokyoBase)).
				PaddingLeft(1)

	tableHeaderBase = lipgloss.NewStyle().
			Background(lipgloss.Color(tokyoSurfaceAlt)).
			Bold(true).
			PaddingLeft(1)

	tableSeparatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(tokyoSubtleText))

	filterBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(tokyoSurfaceAlt)).
			Foreground(lipgloss.Color(tokyoMutedText)).
			Padding(0, 1, 0, 1)

	filterPromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(tokyoMutedText)).Bold(true)
	filterCursorBase  = lipgloss.NewStyle()

	headerTitleBase = lipgloss.NewStyle().
			Bold(true).
			PaddingLeft(1)

	headerTaglineBase = lipgloss.NewStyle().
				Foreground(lipgloss.Color(tokyoAccentBlue)).
				Bold(true).
				PaddingLeft(1)

	headerSubtitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(tokyoMutedText)).
				PaddingLeft(1)

	headerBorderBase = lipgloss.NewStyle().
				Foreground(lipgloss.Color(tokyoSurfaceLine))

	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(tokyoMutedText))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color(tokyoAccentRed)).Bold(true)
	hintStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(tokyoSubtleText))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(tokyoSubtleText))

	modalStyle = lipgloss.NewStyle().
			Padding(1, 3).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(tokyoAccentPurple)).
			Background(lipgloss.Color(tokyoSurface))

	modalContentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(tokyoText)).
				Background(lipgloss.Color(tokyoSurface))

	modalTitleBase = lipgloss.NewStyle().
			Foreground(lipgloss.Color(tokyoAccentPurple)).
			Bold(true)

	modalSubtitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(tokyoMutedText))

	modalStatusBase = lipgloss.NewStyle().
			Foreground(lipgloss.Color(tokyoAccentCyan))

	modalConfirmBase = lipgloss.NewStyle().
				Foreground(lipgloss.Color(tokyoBase)).
				Background(lipgloss.Color(tokyoAccentGreen)).
				Bold(true).
				Padding(0, 1)

	modalCancelBase = lipgloss.NewStyle().
			Foreground(lipgloss.Color(tokyoMutedText)).
			Background(lipgloss.Color(tokyoSurfaceAlt)).
			Padding(0, 1)

	modalActionSpacer = lipgloss.NewStyle().
				Foreground(lipgloss.Color(tokyoSubtleText))

	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(tokyoMutedText))

	helpPanelStyle = lipgloss.NewStyle().
			Padding(1, 3).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(tokyoSurfaceLine)).
			Background(lipgloss.Color(tokyoSurface)).
			Width(56)

	infoToastStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(tokyoAccentCyan))
	successToastStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(tokyoAccentGreen))
	errorToastStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(tokyoAccentRed))
)

const (
	headerLines      = 3
	tableHeaderLines = 1
	footerLines      = 3
	helpLines        = 8
)
