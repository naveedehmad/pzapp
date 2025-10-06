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
	baseDelegate.Styles.SelectedTitle = selectedTitleBase.Background(lipgloss.Color(matrixAccentNeon))
	baseDelegate.Styles.NormalDesc = listDescStyle
	baseDelegate.Styles.SelectedDesc = selectedDescBase.Background(lipgloss.Color(matrixAccentPink))

	model := Model{provider: provider}

	l := list.New([]list.Item{}, baseDelegate, 0, 0)
	l.Title = ""
	l.SetShowTitle(false)
	l.SetShowPagination(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.Styles.HelpStyle = helpStyle
	l.Styles.FilterCursor = filterCursorBase.Foreground(lipgloss.Color(matrixAccentNeon))
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
				return m, nil
			} else if len(m.list.Items()) == 0 {
				// When list is empty (after filtering and killing), refresh to show all ports
				m.list.ResetFilter()
				m.statusMsg = "🔄 Refreshing..."
				cmds = append(cmds, loadPortsCmd(m.provider))
				return m, tea.Batch(cmds...)
			}
			// Let escape fall through to list component to handle search mode exit
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
	
	// Creative protocol indicators
	proto := strings.ToUpper(p.entry.Protocol)
	var protoIcon string
	switch proto {
	case "TCP":
		protoIcon = "🔗"
	case "UDP":
		protoIcon = "📡"
	case "HTTP":
		protoIcon = "🌐"
	case "HTTPS":
		protoIcon = "🔐"
	default:
		protoIcon = "⚡"
	}
	
	// State with cyberpunk indicators
	state := strings.ToLower(p.entry.State)
	var stateIcon string
	switch state {
	case "listening":
		stateIcon = "🎯"
	case "established":
		stateIcon = "🔥"
	case "close_wait":
		stateIcon = "⏳"
	case "time_wait":
		stateIcon = "💭"
	default:
		stateIcon = "🌀"
		if state == "" {
			state = "listening"
			stateIcon = "🎯"
		}
	}
	
	// Port classification with visual indicators
	var portClassIcon string
	port := p.entry.Port
	switch {
	case port < 1024:
		portClassIcon = "👑" // System/privileged ports
	case port >= 1024 && port < 49152:
		portClassIcon = "🎪" // Registered ports
	default:
		portClassIcon = "🎲" // Dynamic/private ports
	}
	
	// Process name with visual enhancement
	process := p.entry.Process
	if state != "" {
		process = fmt.Sprintf("%s [%s]", process, state)
	}

	columns := []string{
		padded(fmt.Sprintf("%s %s", protoIcon, proto), layout.proto),
		padded(fmt.Sprintf("%s %d", portClassIcon, p.entry.Port), layout.port),
		padded(fmt.Sprintf("%s %s", stateIcon, process), layout.process),
		padded(fmt.Sprintf("💀 %d", p.entry.PID), layout.pid),
		padded(fmt.Sprintf("👤 %s", p.entry.User), layout.user),
		padded(fmt.Sprintf("🌍 %s", p.entry.Address), layout.address),
	}
	return "▶ " + strings.Join(columns, " ┃ ")
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
		return matrixAccentNeon
	}
	idx := (m.accentIndex + offset) % len(accentCycle)
	if idx < 0 {
		idx += len(accentCycle)
	}
	return accentCycle[idx]
}

func (m Model) generateMatrixRain(width int) string {
	if width <= 0 {
		return ""
	}
	rain := make([]string, width)
	for i := 0; i < width; i++ {
		if (m.tickCount+i)%7 == 0 {
			charIdx := (m.tickCount + i) % len(matrixChars)
			intensity := (m.tickCount + i) % 4
			var color string
			switch intensity {
			case 0:
				color = matrixAccentGreen
			case 1:
				color = matrixTextDim
			case 2:
				color = matrixTextSubtle
			default:
				color = matrixBorder
			}
			rain[i] = lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(matrixChars[charIdx])
		} else {
			rain[i] = " "
		}
	}
	return strings.Join(rain, "")
}

func (m Model) generateGlitch(text string) string {
	if m.tickCount%20 == 0 && len(text) > 0 {
		runes := []rune(text)
		glitchIdx := m.tickCount % len(runes)
		if glitchIdx < len(glitchChars) {
			runes[glitchIdx] = []rune(glitchChars[m.tickCount%len(glitchChars)])[0]
		}
		return string(runes)
	}
	return text
}

func (m Model) pulseIntensity() float64 {
	return 0.5 + 0.5*float64(m.tickCount%30)/30.0
}

func (m Model) generateDigitalNoise(length int) string {
	if length <= 0 {
		return ""
	}
	noise := make([]string, length)
	for i := 0; i < length; i++ {
		if (m.tickCount+i)%11 == 0 {
			charIdx := (m.tickCount + i*3) % len(glitchChars)
			noise[i] = lipgloss.NewStyle().
				Foreground(lipgloss.Color(matrixTextSubtle)).
				Render(glitchChars[charIdx])
		} else {
			noise[i] = " "
		}
	}
	return strings.Join(noise, "")
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
	accentQuad := lipgloss.Color(m.accentColor(3))

	// Matrix rain effect at top
	matrixRain := m.generateMatrixRain(m.width)
	
	// Epic title with digital effects
	titleText := "██████╗ ███████╗ █████╗ ██████╗ ██████╗ \n██╔══██╗╚══███╔╝██╔══██╗██╔══██╗██╔══██╗\n██████╔╝  ███╔╝ ███████║██████╔╝██████╔╝\n██╔═══╝  ███╔╝  ██╔══██║██╔═══╝ ██╔═══╝ \n██║     ███████╗██║  ██║██║     ██║     \n╚═╝     ╚══════╝╚═╝  ╚═╝╚═╝     ╚═╝     "
	title := lipgloss.NewStyle().
		Foreground(accentPrimary).
		Bold(true).
		Render(titleText)

	// Animated tagline with glitch effect
	taglineText := "⚡ NEURAL NETWORK PORT SCANNER ⚡"
	if len(taglineCycle) > 0 {
		taglineText = taglineCycle[m.taglineIndex%len(taglineCycle)]
	}
	glitchedTagline := m.generateGlitch(taglineText)
	tagline := headerTaglineBase.Foreground(accentSecondary).Render(glitchedTagline)
	
	// System status with cyberpunk flair
	statusLine := fmt.Sprintf("【 QUANTUM CORE ACTIVE 】【 %d TARGETS ACQUIRED 】【 MATRIX SYNCHRONIZED 】", len(m.list.Items()))
	systemStatus := headerSubtitleStyle.Foreground(accentTertiary).Render(statusLine)
	
	// Dynamic border with digital noise
	borderChars := []string{"═", "━", "▬", "⬛", "▪", "▫", "░", "▒", "▓"}
	borderChar := borderChars[m.tickCount%len(borderChars)]
	border := headerBorderBase.Foreground(accentQuad).Render(strings.Repeat(borderChar, max(0, m.width)))
	
	// Digital noise line
	digitalNoise := m.generateDigitalNoise(m.width)

	return lipgloss.JoinVertical(lipgloss.Left,
		matrixRain,
		lipgloss.PlaceHorizontal(m.width, lipgloss.Center, title),
		lipgloss.PlaceHorizontal(m.width, lipgloss.Center, tagline),
		lipgloss.PlaceHorizontal(m.width, lipgloss.Center, systemStatus),
		border,
		digitalNoise,
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
		matrixAccentGold,
		matrixAccentGreen,
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
	
	// Status with cyberpunk styling
	if m.errMsg != "" {
		errorMsg := fmt.Sprintf("【 ⚠️  SYSTEM ERROR ⚠️  】 %s", m.errMsg)
		lines = append(lines, errorStyle.Render(errorMsg))
	} else if m.statusMsg != "" {
		statusMsg := fmt.Sprintf("【 ⚡ STATUS ⚡ 】 %s", m.statusMsg)
		lines = append(lines, statusStyle.Render(statusMsg))
	}

	if m.toast.message != "" {
		lines = append(lines, toastStyle(m.toast))
	}

	// Enhanced control hints with cyberpunk aesthetics
	controlSections := []string{
		"🎯 j/k/↑↓ NAVIGATE",
		"💀 enter/d TERMINATE",
		"🔄 r REFRESH",
		"🔍 / SCAN",
		"❓ ? INFO",
		"💨 q ESCAPE",
	}
	controlHint := strings.Join(controlSections, "  ║  ")
	hint := hintStyle.Render(fmt.Sprintf("【 COMMAND MATRIX 】 %s", controlHint))
	lines = append(lines, hint)
	
	// Digital noise footer
	digitalFooter := m.generateDigitalNoise(m.width)
	lines = append(lines, digitalFooter)

	rendered := make([]string, len(lines))
	for i, line := range lines {
		rendered[i] = lipgloss.PlaceHorizontal(m.width, lipgloss.Left, line)
	}
	return strings.Join(rendered, "\n")
}

func renderKillModal(entry ports.Port, inFlight bool, width int) string {
	// Epic ASCII art warning
	warningArt := `
    ███████╗██╗    ██╗ █████╗ ██████╗ ███╗   ██╗██╗███╗   ██╗ ██████╗ 
    ██╔════╝██║    ██║██╔══██╗██╔══██╗████╗  ██║██║████╗  ██║██╔════╝ 
    ███████╗██║ █╗ ██║███████║██████╔╝██╔██╗ ██║██║██╔██╗ ██║██║  ███╗
    ╚════██║██║███╗██║██╔══██║██╔══██╗██║╚██╗██║██║██║╚██╗██║██║   ██║
    ███████║╚███╔███╔╝██║  ██║██║  ██║██║ ╚████║██║██║ ╚████║╚██████╔╝
    ╚══════╝ ╚══╝╚══╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝╚═╝  ╚═══╝ ╚═════╝ `
	
	subtitle := fmt.Sprintf("【 TARGET ACQUIRED 】 %s | %s:%d | PID:%d", 
		entry.Process, strings.ToUpper(entry.Protocol), entry.Port, entry.PID)
	
	var status string
	if inFlight {
		status = "🔥💀 QUANTUM TERMINATION SEQUENCE INITIATED 💀🔥\n⚡⚡⚡ NEURAL PATHWAYS SEVERING ⚡⚡⚡"
	} else {
		status = "💀 INITIATE DIGITAL ANNIHILATION PROTOCOL? 💀\n⚔️  WARNING: PROCESS WILL BE ELIMINATED ⚔️"
	}

	modalWidth := clamp(width-4, 40, 80)
	innerWidth := modalWidth - modalStyle.GetPaddingLeft() - modalStyle.GetPaddingRight()
	if innerWidth < 20 {
		innerWidth = 20
	}

	lines := []string{
		modalTitleBase.Render(warningArt),
		"",
		modalSubtitleStyle.Render(subtitle),
		"",
		modalStatusBase.Render(status),
		"",
		lipgloss.JoinHorizontal(lipgloss.Left,
			modalConfirmBase.Render("💀⚔️  [Y] EXECUTE TERMINATION"),
			modalActionSpacer.Render("    "),
			modalCancelBase.Render("🛡️  [N] ABORT MISSION"),
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
	helpHeader := `
    ██╗  ██╗███████╗██╗     ██████╗     ███╗   ███╗ █████╗ ████████╗██████╗ ██╗██╗  ██╗
    ██║  ██║██╔════╝██║     ██╔══██╗    ████╗ ████║██╔══██╗╚══██╔══╝██╔══██╗██║╚██╗██╔╝
    ███████║█████╗  ██║     ██████╔╝    ██╔████╔██║███████║   ██║   ██████╔╝██║ ╚███╔╝ 
    ██╔══██║██╔══╝  ██║     ██╔═══╝     ██║╚██╔╝██║██╔══██║   ██║   ██╔══██╗██║ ██╔██╗ 
    ██║  ██║███████╗███████╗██║         ██║ ╚═╝ ██║██║  ██║   ██║   ██║  ██║██║██╔╝ ██╗
    ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝         ╚═╝     ╚═╝╚═╝  ╚═╝   ╚═╝   ╚═╝  ╚═╝╚═╝╚═╝  ╚═╝`

	commandSections := [][]string{
		{"【 NAVIGATION PROTOCOLS 】", "", ""},
		{"🎯 Movement", "j/k ↑↓", "Navigate through targets"},
		{"🎯 Selection", "enter", "Select current target"},
		{"", "", ""},
		{"【 COMBAT OPERATIONS 】", "", ""},
		{"💀 Terminate", "d/enter", "Execute termination protocol"},
		{"⚔️  Confirm", "y/Y", "Confirm elimination"},
		{"🛡️  Abort", "n/N/esc", "Abort current operation"},
		{"", "", ""},
		{"【 SYSTEM OPERATIONS 】", "", ""},
		{"🔄 Refresh", "r", "Reload target matrix"},
		{"🔍 Scan", "/", "Initiate search protocol"},
		{"💨 Escape", "esc", "Exit search mode"},
		{"❓ Info", "?", "Toggle command matrix"},
		{"💨 Logout", "q/ctrl+c", "Exit system"},
	}

	rows := make([]string, 0, len(commandSections))
	for _, section := range commandSections {
		if section[0] == "" {
			rows = append(rows, "")
		} else if section[1] == "" {
			// Header section
			headerStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(matrixAccentNeon)).
				Bold(true)
			rows = append(rows, headerStyle.Render(section[0]))
		} else {
			// Command row
			commandStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(matrixText))
			keyStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(matrixAccentPink)).
				Bold(true)
			descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(matrixTextDim))
			
			row := fmt.Sprintf("%-20s %-12s %s",
				commandStyle.Render(section[0]),
				keyStyle.Render(section[1]),
				descStyle.Render(section[2]))
			rows = append(rows, row)
		}
	}

	headerStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color(matrixAccentGold)).
		Render(helpHeader)
	
	body := lipgloss.JoinVertical(lipgloss.Left, 
		headerStyled,
		"",
		strings.Join(rows, "\n"))
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
	// Matrix/Cyberpunk Color Palette
	matrixBlack       = "#000000"
	matrixDeepDark    = "#0a0a0a"
	matrixDark        = "#0f0f23"
	matrixSurface     = "#1a1a2e"
	matrixPanel       = "#16213e"
	matrixBorder      = "#0f3460"
	matrixText        = "#00ff41"
	matrixTextDim     = "#00cc33"
	matrixTextSubtle  = "#009933"
	matrixAccentNeon  = "#00ffff"
	matrixAccentPink  = "#ff007f"
	matrixAccentPurple= "#9d4edd"
	matrixAccentBlue  = "#0077be"
	matrixAccentGreen = "#39ff14"
	matrixAccentRed   = "#ff073a"
	matrixAccentGold  = "#ffd700"
	matrixWarning     = "#ffff00"
)

var accentCycle = []string{
	matrixAccentNeon,
	matrixAccentPink,
	matrixAccentPurple,
	matrixAccentBlue,
	matrixAccentGreen,
	matrixAccentGold,
}

var matrixChars = []string{"0", "1", "╫", "╬", "│", "┤", "┐", "└", "┴", "┬", "├", "─", "┼", "╭", "╮", "╯", "╰"}

var glitchChars = []string{"█", "▉", "▊", "▋", "▌", "▍", "▎", "▏", "░", "▒", "▓", "■", "□", "▪", "▫"}

var taglineCycle = []string{
	"⚡ NEURAL NETWORK PORT SCANNER ⚡",
	"🔥 MATRIX PROTOCOL TERMINATOR 🔥", 
	"🌊 DIGITAL OCEAN NAVIGATOR 🌊",
	"⚔️  CYBER WARFARE SPECIALIST ⚔️",
	"🛸 QUANTUM PORT DECIMATOR 🛸",
	"💀 GHOST IN THE SHELL 💀",
	"🎯 PRECISION STRIKE SYSTEM 🎯",
	"🌌 INTERDIMENSIONAL GATEWAY 🌌",
}

var (
	listTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(matrixText)).
			Background(lipgloss.Color(matrixSurface)).
			PaddingLeft(1)

	listDescStyle = listTitleStyle.Copy().
			Foreground(lipgloss.Color(matrixTextDim))

	selectedTitleBase = lipgloss.NewStyle().
				Foreground(lipgloss.Color(matrixBlack)).
				Bold(true).
				PaddingLeft(1)

	selectedDescBase = lipgloss.NewStyle().
				Foreground(lipgloss.Color(matrixBlack)).
				PaddingLeft(1)

	tableHeaderBase = lipgloss.NewStyle().
			Background(lipgloss.Color(matrixPanel)).
			Bold(true).
			PaddingLeft(1)

	tableSeparatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(matrixTextSubtle))

	filterBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(matrixPanel)).
			Foreground(lipgloss.Color(matrixTextDim)).
			Padding(0, 1, 0, 1).
			Border(lipgloss.DoubleBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color(matrixAccentNeon))

	filterPromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(matrixAccentNeon)).Bold(true)
	filterCursorBase  = lipgloss.NewStyle()

	headerTitleBase = lipgloss.NewStyle().
			Bold(true).
			PaddingLeft(1)

	headerTaglineBase = lipgloss.NewStyle().
				Foreground(lipgloss.Color(matrixAccentNeon)).
				Bold(true).
				PaddingLeft(1)

	headerSubtitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(matrixTextDim)).
				PaddingLeft(1)

	headerBorderBase = lipgloss.NewStyle().
				Foreground(lipgloss.Color(matrixBorder))

	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(matrixTextDim))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color(matrixAccentRed)).Bold(true)
	hintStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(matrixTextSubtle))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(matrixTextSubtle))

	modalStyle = lipgloss.NewStyle().
			Padding(1, 3).
			Border(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color(matrixAccentPink)).
			Background(lipgloss.Color(matrixSurface))

	modalContentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(matrixText)).
				Background(lipgloss.Color(matrixSurface))

	modalTitleBase = lipgloss.NewStyle().
			Foreground(lipgloss.Color(matrixAccentPink)).
			Bold(true)

	modalSubtitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(matrixTextDim))

	modalStatusBase = lipgloss.NewStyle().
			Foreground(lipgloss.Color(matrixAccentNeon))

	modalConfirmBase = lipgloss.NewStyle().
				Foreground(lipgloss.Color(matrixBlack)).
				Background(lipgloss.Color(matrixAccentGreen)).
				Bold(true).
				Padding(0, 1)

	modalCancelBase = lipgloss.NewStyle().
			Foreground(lipgloss.Color(matrixTextDim)).
			Background(lipgloss.Color(matrixPanel)).
			Padding(0, 1)

	modalActionSpacer = lipgloss.NewStyle().
				Foreground(lipgloss.Color(matrixTextSubtle))

	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(matrixTextDim))

	helpPanelStyle = lipgloss.NewStyle().
			Padding(1, 3).
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color(matrixBorder)).
			Background(lipgloss.Color(matrixSurface)).
			Width(60)

	infoToastStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(matrixAccentNeon))
	successToastStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(matrixAccentGreen))
	errorToastStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(matrixAccentRed))
)

const (
	headerLines      = 10
	tableHeaderLines = 1
	footerLines      = 5
	helpLines        = 15
)
