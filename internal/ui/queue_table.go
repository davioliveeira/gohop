package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/davioliveeira/gohop/internal/config"
	"github.com/davioliveeira/gohop/internal/rabbitmq"
)

// ═══════════════════════════════════════════════════════════════════════════════
// MODELO DA TABELA DE FILAS
// ═══════════════════════════════════════════════════════════════════════════════

type queueTableModel struct {
	queues       []rabbitmq.QueueInfoManagement
	filteredQ    []rabbitmq.QueueInfoManagement
	cursor       int
	sortColumn   int
	sortDesc     bool
	loading      bool
	quitting     bool
	width        int
	height       int
	offset       int // Para scroll
	filter       string
	spinner      spinner.Model
	animPhase    float64
	showDetails  bool
	cfg          *config.Config
}

// Colunas da tabela
type column struct {
	title string
	width int
	align lipgloss.Position
}

var tableColumns = []column{
	{title: "FILA", width: 32, align: lipgloss.Left},
	{title: "TIPO", width: 10, align: lipgloss.Center},
	{title: "READY", width: 10, align: lipgloss.Right},
	{title: "UNACKED", width: 10, align: lipgloss.Right},
	{title: "TOTAL", width: 10, align: lipgloss.Right},
	{title: "CONSUMERS", width: 10, align: lipgloss.Center},
	{title: "STATUS", width: 14, align: lipgloss.Center},
}

// Mensagens
type tableTickMsg time.Time
type tableLoadedMsg struct {
	queues []rabbitmq.QueueInfoManagement
	err    error
}

// ═══════════════════════════════════════════════════════════════════════════════
// CONSTRUTOR
// ═══════════════════════════════════════════════════════════════════════════════

func NewQueueTableModel(cfg *config.Config) queueTableModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(PrimaryColor)

	return queueTableModel{
		loading:    true,
		sortColumn: 0,
		sortDesc:   false,
		spinner:    s,
		cfg:        cfg,
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// BUBBLE TEA INTERFACE
// ═══════════════════════════════════════════════════════════════════════════════

func (m queueTableModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadQueues,
		tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
			return tableTickMsg(t)
		}),
	)
}

func (m queueTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}

		case "down", "j":
			if m.cursor < len(m.filteredQ)-1 {
				m.cursor++
				maxVisible := m.getMaxVisibleRows()
				if m.cursor >= m.offset+maxVisible {
					m.offset = m.cursor - maxVisible + 1
				}
			}

		case "home", "g":
			m.cursor = 0
			m.offset = 0

		case "end", "G":
			m.cursor = len(m.filteredQ) - 1
			maxVisible := m.getMaxVisibleRows()
			if m.cursor >= maxVisible {
				m.offset = m.cursor - maxVisible + 1
			}

		case "1": // Ordenar por Nome
			m.sortBy(0)
		case "2": // Ordenar por Tipo
			m.sortBy(1)
		case "3": // Ordenar por Ready
			m.sortBy(2)
		case "4": // Ordenar por Unacked
			m.sortBy(3)
		case "5": // Ordenar por Total
			m.sortBy(4)
		case "6": // Ordenar por Consumers
			m.sortBy(5)

		case "r", "R": // Refresh
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.loadQueues)

		case "enter", " ": // Toggle detalhes
			m.showDetails = !m.showDetails

		case "pgup":
			maxVisible := m.getMaxVisibleRows()
			m.cursor -= maxVisible
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.offset = m.cursor

		case "pgdown":
			maxVisible := m.getMaxVisibleRows()
			m.cursor += maxVisible
			if m.cursor >= len(m.filteredQ) {
				m.cursor = len(m.filteredQ) - 1
			}
			if m.cursor >= m.offset+maxVisible {
				m.offset = m.cursor - maxVisible + 1
			}
		}

	case tableTickMsg:
		m.animPhase += 0.1
		return m, tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
			return tableTickMsg(t)
		})

	case tableLoadedMsg:
		m.loading = false
		if msg.err != nil {
			// Erro será mostrado na view
		} else {
			m.queues = msg.queues
			m.filteredQ = msg.queues
			m.sortBy(m.sortColumn)
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m queueTableModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Header
	header := m.renderHeader()
	b.WriteString(header)
	b.WriteString("\n")

	if m.loading {
		// Loading state
		loadingBox := m.renderLoadingBox()
		b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, loadingBox))
	} else if len(m.filteredQ) == 0 {
		// Empty state
		emptyBox := m.renderEmptyBox()
		b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, emptyBox))
	} else {
		// Tabela
		table := m.renderTable()
		b.WriteString(table)

		// Detalhes da fila selecionada
		if m.showDetails && m.cursor < len(m.filteredQ) {
			b.WriteString("\n")
			details := m.renderDetails(m.filteredQ[m.cursor])
			b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, details))
		}
	}

	b.WriteString("\n")

	// Footer
	footer := m.renderFooter()
	b.WriteString(footer)

	return b.String()
}

// ═══════════════════════════════════════════════════════════════════════════════
// RENDERIZAÇÃO
// ═══════════════════════════════════════════════════════════════════════════════

func (m queueTableModel) renderHeader() string {
	// Logo estilizado
	logoStyle := lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true)

	logo := logoStyle.Render("🐰 GoHop")

	// Título
	titleStyle := lipgloss.NewStyle().
		Foreground(TextPrimary).
		Bold(true)
	title := titleStyle.Render(" Queue Explorer")

	// Contador
	countStyle := lipgloss.NewStyle().
		Foreground(AccentColor).
		Bold(true)
	count := ""
	if !m.loading {
		count = countStyle.Render(fmt.Sprintf(" (%d filas)", len(m.filteredQ)))
	}

	headerLine := logo + title + count

	// Separador decorativo
	sepStyle := lipgloss.NewStyle().Foreground(MutedColorDark)
	separator := sepStyle.Render(strings.Repeat("━", 90))

	return lipgloss.JoinVertical(lipgloss.Center,
		"",
		headerLine,
		separator,
	)
}

func (m queueTableModel) renderLoadingBox() string {
	loadingStyle := lipgloss.NewStyle().
		Foreground(InfoColor).
		Align(lipgloss.Center)

	content := fmt.Sprintf("\n%s Carregando filas...\n\n", m.spinner.View())

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Padding(2, 4).
		Width(50).
		Render(loadingStyle.Render(content))
}

func (m queueTableModel) renderEmptyBox() string {
	emptyStyle := lipgloss.NewStyle().
		Foreground(MutedColor).
		Align(lipgloss.Center)

	content := "\n📭 Nenhuma fila encontrada\n\nVerifique a conexão com o broker\n"

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(MutedColor).
		Padding(2, 4).
		Width(50).
		Render(emptyStyle.Render(content))
}

func (m queueTableModel) renderTable() string {
	var rows []string

	// Header da tabela
	headerRow := m.renderTableHeader()
	rows = append(rows, headerRow)

	// Separador
	sepStyle := lipgloss.NewStyle().Foreground(MutedColorDark)
	rows = append(rows, sepStyle.Render(strings.Repeat("─", 98)))

	// Linhas de dados
	maxVisible := m.getMaxVisibleRows()
	endIdx := m.offset + maxVisible
	if endIdx > len(m.filteredQ) {
		endIdx = len(m.filteredQ)
	}

	for i := m.offset; i < endIdx; i++ {
		q := m.filteredQ[i]
		isSelected := i == m.cursor
		row := m.renderTableRow(q, isSelected, i)
		rows = append(rows, row)
	}

	// Indicador de scroll se necessário
	if len(m.filteredQ) > maxVisible {
		scrollInfo := m.renderScrollInfo()
		rows = append(rows, "")
		rows = append(rows, scrollInfo)
	}

	table := lipgloss.JoinVertical(lipgloss.Left, rows...)

	// Box da tabela
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(SecondaryColor).
		Padding(0, 1).
		Render(table)
}

func (m queueTableModel) renderTableHeader() string {
	var cells []string

	for i, col := range tableColumns {
		style := lipgloss.NewStyle().
			Width(col.width).
			Align(col.align).
			Foreground(AccentColor).
			Bold(true)

		// Indicador de ordenação
		title := col.title
		if i == m.sortColumn {
			if m.sortDesc {
				title = col.title + " ▼"
			} else {
				title = col.title + " ▲"
			}
			style = style.Foreground(PrimaryColor)
		}

		cells = append(cells, style.Render(title))
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, cells...)
}

func (m queueTableModel) renderTableRow(q rabbitmq.QueueInfoManagement, selected bool, idx int) string {
	var cells []string

	// Estilo base
	baseStyle := lipgloss.NewStyle()
	if selected {
		baseStyle = baseStyle.
			Background(lipgloss.Color("#2C323C")).
			Bold(true)
	}

	// Nome (com ícone de status)
	statusIcon := m.getStatusIcon(q)
	nameStyle := baseStyle.Copy().
		Width(tableColumns[0].width).
		Align(tableColumns[0].align).
		Foreground(TextPrimary)
	if selected {
		nameStyle = nameStyle.Foreground(PrimaryColor)
	}
	cells = append(cells, nameStyle.Render(statusIcon+" "+truncateString(q.Name, 28)))

	// Tipo
	typeColor := InfoColor
	if q.Type == "quorum" {
		typeColor = SuccessColor
	}
	typeStyle := baseStyle.Copy().
		Width(tableColumns[1].width).
		Align(tableColumns[1].align).
		Foreground(typeColor)
	cells = append(cells, typeStyle.Render(q.Type))

	// Ready
	readyColor := m.getMessageColor(q.MessagesReady)
	readyStyle := baseStyle.Copy().
		Width(tableColumns[2].width).
		Align(tableColumns[2].align).
		Foreground(readyColor).
		Bold(q.MessagesReady > 0)
	cells = append(cells, readyStyle.Render(formatNumber(q.MessagesReady)))

	// Unacked
	unackedColor := MutedColor
	if q.MessagesUnacked > 0 {
		unackedColor = WarningColor
	}
	unackedStyle := baseStyle.Copy().
		Width(tableColumns[3].width).
		Align(tableColumns[3].align).
		Foreground(unackedColor)
	cells = append(cells, unackedStyle.Render(formatNumber(q.MessagesUnacked)))

	// Total
	totalStyle := baseStyle.Copy().
		Width(tableColumns[4].width).
		Align(tableColumns[4].align).
		Foreground(InfoColor)
	cells = append(cells, totalStyle.Render(formatNumber(q.Messages)))

	// Consumers
	consumerColor := ErrorColor
	if q.Consumers > 0 {
		consumerColor = SuccessColor
	}
	consumerStyle := baseStyle.Copy().
		Width(tableColumns[5].width).
		Align(tableColumns[5].align).
		Foreground(consumerColor).
		Bold(true)
	cells = append(cells, consumerStyle.Render(fmt.Sprintf("%d", q.Consumers)))

	// Status
	status := m.getQueueStatusText(q)
	statusStyle := baseStyle.Copy().
		Width(tableColumns[6].width).
		Align(tableColumns[6].align)
	cells = append(cells, statusStyle.Render(status))

	return lipgloss.JoinHorizontal(lipgloss.Left, cells...)
}

func (m queueTableModel) renderScrollInfo() string {
	current := m.cursor + 1
	total := len(m.filteredQ)
	percent := float64(current) / float64(total) * 100

	// Barra de scroll visual
	barWidth := 20
	filled := int(float64(barWidth) * float64(current) / float64(total))
	if filled < 1 {
		filled = 1
	}

	bar := lipgloss.NewStyle().Foreground(PrimaryColor).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(MutedColorDark).Render(strings.Repeat("░", barWidth-filled))

	info := lipgloss.NewStyle().Foreground(MutedColor).Render(
		fmt.Sprintf(" %d/%d (%.0f%%)", current, total, percent))

	return lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(98).
		Render(bar + info)
}

func (m queueTableModel) renderDetails(q rabbitmq.QueueInfoManagement) string {
	// Painel de detalhes expandido
	titleStyle := lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(MutedColor).
		Width(20)

	valueStyle := lipgloss.NewStyle().
		Foreground(TextPrimary).
		Bold(true)

	var lines []string
	lines = append(lines, titleStyle.Render("📋 DETALHES DA FILA"))
	lines = append(lines, "")

	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("Nome completo:"),
		valueStyle.Render(q.Name)))

	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("Tipo:"),
		valueStyle.Copy().Foreground(InfoColor).Render(q.Type)))

	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("VHost:"),
		valueStyle.Render(q.VHost)))

	lines = append(lines, "")

	// Métricas
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("📥 Messages Ready:"),
		valueStyle.Copy().Foreground(m.getMessageColor(q.MessagesReady)).Render(formatNumber(q.MessagesReady))))

	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("⏳ Unacknowledged:"),
		valueStyle.Copy().Foreground(WarningColor).Render(formatNumber(q.MessagesUnacked))))

	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("📊 Total:"),
		valueStyle.Copy().Foreground(InfoColor).Render(formatNumber(q.Messages))))

	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("👥 Consumers:"),
		valueStyle.Copy().Foreground(SuccessColor).Render(fmt.Sprintf("%d", q.Consumers))))

	// Barra visual
	if q.Messages > 0 {
		lines = append(lines, "")
		lines = append(lines, m.renderVisualBar(q.Messages, 1000, 40))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(AccentColor).
		Padding(1, 2).
		Width(60).
		Render(content)
}

func (m queueTableModel) renderVisualBar(current, max, width int) string {
	if max == 0 {
		max = 1
	}

	percent := float64(current) / float64(max)
	if percent > 1.0 {
		percent = 1.0
	}

	filled := int(float64(width) * percent)
	empty := width - filled

	var barColor lipgloss.Color
	if percent < 0.3 {
		barColor = SuccessColor
	} else if percent < 0.7 {
		barColor = WarningColor
	} else {
		barColor = ErrorColor
	}

	filledBar := lipgloss.NewStyle().Foreground(barColor).Render(strings.Repeat("█", filled))
	emptyBar := lipgloss.NewStyle().Foreground(MutedColorDark).Render(strings.Repeat("░", empty))

	return filledBar + emptyBar
}

func (m queueTableModel) renderFooter() string {
	// Atalhos de teclado
	keyStyle := lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(MutedColor)
	sepStyle := lipgloss.NewStyle().Foreground(MutedColorDark)

	shortcuts := []string{
		keyStyle.Render("↑↓") + descStyle.Render(" navegar"),
		keyStyle.Render("1-6") + descStyle.Render(" ordenar"),
		keyStyle.Render("enter") + descStyle.Render(" detalhes"),
		keyStyle.Render("r") + descStyle.Render(" atualizar"),
		keyStyle.Render("q") + descStyle.Render(" sair"),
	}

	shortcutsLine := strings.Join(shortcuts, sepStyle.Render("  │  "))

	// Dica de ordenação
	sortHint := descStyle.Render("Ordenar: ") +
		keyStyle.Render("1") + descStyle.Render("-Nome ") +
		keyStyle.Render("2") + descStyle.Render("-Tipo ") +
		keyStyle.Render("3") + descStyle.Render("-Ready ") +
		keyStyle.Render("4") + descStyle.Render("-Unacked ") +
		keyStyle.Render("5") + descStyle.Render("-Total ") +
		keyStyle.Render("6") + descStyle.Render("-Consumers")

	footer := lipgloss.JoinVertical(lipgloss.Center, shortcutsLine, sortHint)

	return lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Padding(1, 0).
		Render(footer)
}

// ═══════════════════════════════════════════════════════════════════════════════
// HELPERS
// ═══════════════════════════════════════════════════════════════════════════════

func (m queueTableModel) getMaxVisibleRows() int {
	// Calcular baseado na altura disponível
	availableHeight := m.height - 12 // Header, footer, bordas
	if availableHeight < 5 {
		return 5
	}
	if availableHeight > 25 {
		return 25
	}
	return availableHeight
}

func (m queueTableModel) getStatusIcon(q rabbitmq.QueueInfoManagement) string {
	if q.Consumers == 0 && q.Messages > 0 {
		return "⚠"
	}
	if q.Messages > 1000 {
		return "🔴"
	}
	if q.Messages > 100 {
		return "🟡"
	}
	if q.Messages > 0 {
		return "🟢"
	}
	return "⚪"
}

func (m queueTableModel) getQueueStatusText(q rabbitmq.QueueInfoManagement) string {
	if q.Consumers == 0 && q.Messages > 0 {
		return lipgloss.NewStyle().Foreground(ErrorColor).Bold(true).Render("⚠ SEM CONSUMER")
	}
	if q.Messages > 1000 {
		return lipgloss.NewStyle().Foreground(ErrorColor).Render("🔴 ALTO")
	}
	if q.Messages > 100 {
		return lipgloss.NewStyle().Foreground(WarningColor).Render("🟡 MODERADO")
	}
	if q.Messages > 0 {
		return lipgloss.NewStyle().Foreground(SuccessColor).Render("🟢 NORMAL")
	}
	return lipgloss.NewStyle().Foreground(MutedColor).Render("⚪ VAZIA")
}

func (m queueTableModel) getMessageColor(count int) lipgloss.Color {
	if count > 1000 {
		return ErrorColor
	}
	if count > 100 {
		return WarningColor
	}
	if count > 0 {
		return SuccessColor
	}
	return MutedColor
}

func (m *queueTableModel) sortBy(col int) {
	if m.sortColumn == col {
		m.sortDesc = !m.sortDesc
	} else {
		m.sortColumn = col
		m.sortDesc = false
	}

	sort.Slice(m.filteredQ, func(i, j int) bool {
		var less bool
		switch col {
		case 0: // Nome
			less = m.filteredQ[i].Name < m.filteredQ[j].Name
		case 1: // Tipo
			less = m.filteredQ[i].Type < m.filteredQ[j].Type
		case 2: // Ready
			less = m.filteredQ[i].MessagesReady < m.filteredQ[j].MessagesReady
		case 3: // Unacked
			less = m.filteredQ[i].MessagesUnacked < m.filteredQ[j].MessagesUnacked
		case 4: // Total
			less = m.filteredQ[i].Messages < m.filteredQ[j].Messages
		case 5: // Consumers
			less = m.filteredQ[i].Consumers < m.filteredQ[j].Consumers
		default:
			less = false
		}

		if m.sortDesc {
			return !less
		}
		return less
	})
}

func (m queueTableModel) loadQueues() tea.Msg {
	mgmtClient := rabbitmq.NewManagementClient(m.cfg.RabbitMQ)
	queues, err := mgmtClient.ListQueues()
	return tableLoadedMsg{queues: queues, err: err}
}

func formatNumber(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

// ═══════════════════════════════════════════════════════════════════════════════
// EXECUÇÃO
// ═══════════════════════════════════════════════════════════════════════════════

func RunQueueTable(cfg *config.Config) error {
	if !isTerminal() {
		return fmt.Errorf("terminal não interativo")
	}

	model := NewQueueTableModel(cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())

	_, err := p.Run()
	return err
}
