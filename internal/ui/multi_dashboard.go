package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/davioliveeira/gohop/internal/config"
	"github.com/davioliveeira/gohop/internal/rabbitmq"
)

// MultiQueueData representa dados de múltiplas filas
type MultiQueueData struct {
	Queues []QueueData
}

// MultiDashboardModel representa o dashboard de múltiplas filas
type MultiDashboardModel struct {
	queueNames []string
	cfg        *config.Config
	queuesData []QueueData
	spinner    spinner.Model
	loading    bool
	error      string
	interval   time.Duration
	lastUpdate time.Time
	width      int
	height     int
	selected   int // Fila selecionada para highlight
}

// multiTickMsg é enviado periodicamente
type multiTickMsg struct {
	t time.Time
}

// multiUpdateMsg contém dados atualizados
type multiUpdateMsg struct {
	queuesData []QueueData
	err        error
}

// NewMultiDashboard cria um novo dashboard para múltiplas filas
func NewMultiDashboard(queueNames []string, cfg *config.Config, interval time.Duration) MultiDashboardModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(PrimaryColor)

	return MultiDashboardModel{
		queueNames: queueNames,
		cfg:        cfg,
		spinner:    s,
		loading:    true,
		interval:   interval,
		selected:   0,
	}
}

// Init inicializa o modelo
func (m MultiDashboardModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.fetchAllData,
		multiTick(),
	)
}

// Update processa mensagens
func (m MultiDashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r", "R":
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.fetchAllData)
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.queuesData)-1 {
				m.selected++
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case multiTickMsg:
		m.loading = true
		return m, tea.Batch(
			m.spinner.Tick,
			m.fetchAllData,
			multiTickAfter(m.interval),
		)

	case multiUpdateMsg:
		m.loading = false
		m.lastUpdate = time.Now()
		if msg.err != nil {
			m.error = msg.err.Error()
		} else {
			m.error = ""
			m.queuesData = msg.queuesData
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renderiza o dashboard
func (m MultiDashboardModel) View() string {
	var b strings.Builder

	// Header
	header := m.renderHeader()
	b.WriteString(header)
	b.WriteString("\n\n")

	// Loading ou erro
	if m.loading && len(m.queuesData) == 0 {
		loadingMsg := fmt.Sprintf("%s Carregando dados das filas...", m.spinner.View())
		b.WriteString(lipgloss.NewStyle().
			Foreground(InfoColor).
			Align(lipgloss.Center).
			Width(m.width).
			Render(loadingMsg))
		b.WriteString("\n")
	} else if m.error != "" {
		errorMsg := fmt.Sprintf("❌ Erro: %s", m.error)
		b.WriteString(lipgloss.NewStyle().
			Foreground(ErrorColor).
			Align(lipgloss.Center).
			Width(m.width).
			Render(errorMsg))
		b.WriteString("\n")
	} else {
		// Tabela de filas
		table := m.renderTable()
		b.WriteString(table)
		b.WriteString("\n")

		// Resumo
		summary := m.renderSummary()
		b.WriteString(summary)
	}

	// Footer
	b.WriteString("\n")
	footer := m.renderFooter()
	b.WriteString(footer)

	return b.String()
}

func (m MultiDashboardModel) renderHeader() string {
	title := lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true).
		Render("📊 Monitor de Múltiplas Filas")

	subtitle := lipgloss.NewStyle().
		Foreground(MutedColor).
		Render(fmt.Sprintf("Monitorando %d filas", len(m.queueNames)))

	// Status de atualização
	var status string
	if m.loading {
		status = lipgloss.NewStyle().
			Foreground(WarningColor).
			Render(fmt.Sprintf("%s Atualizando...", m.spinner.View()))
	} else {
		status = lipgloss.NewStyle().
			Foreground(SuccessColor).
			Render(fmt.Sprintf("✓ Atualizado: %s", m.lastUpdate.Format("15:04:05")))
	}

	header := lipgloss.JoinVertical(lipgloss.Center, title, subtitle, status)
	return lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(header)
}

func (m MultiDashboardModel) renderTable() string {
	if len(m.queuesData) == 0 {
		return lipgloss.NewStyle().
			Foreground(MutedColor).
			Italic(true).
			Render("Nenhuma fila encontrada")
	}

	// Estilos
	headerStyle := lipgloss.NewStyle().
		Foreground(AccentColor).
		Bold(true).
		Padding(0, 1)

	normalRow := lipgloss.NewStyle().
		Foreground(TextPrimary).
		Padding(0, 1)

	selectedRow := lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true).
		Background(lipgloss.Color("#2C323C")).
		Padding(0, 1)

	// Header da tabela
	headers := []string{
		headerStyle.Width(30).Render("Fila"),
		headerStyle.Width(10).Align(lipgloss.Right).Render("Ready"),
		headerStyle.Width(10).Align(lipgloss.Right).Render("Unacked"),
		headerStyle.Width(10).Align(lipgloss.Right).Render("Total"),
		headerStyle.Width(10).Align(lipgloss.Right).Render("Consumers"),
		headerStyle.Width(12).Render("Status"),
	}
	headerRow := lipgloss.JoinHorizontal(lipgloss.Left, headers...)

	// Separador
	separator := lipgloss.NewStyle().
		Foreground(MutedColorDark).
		Render(strings.Repeat("─", 85))

	var rows []string
	rows = append(rows, headerRow)
	rows = append(rows, separator)

	// Linhas de dados
	for i, q := range m.queuesData {
		style := normalRow
		if i == m.selected {
			style = selectedRow
		}

		// Status com cor
		status := m.getQueueStatus(q)

		row := []string{
			style.Width(30).Render(truncateString(q.Name, 28)),
			style.Width(10).Align(lipgloss.Right).Render(fmt.Sprintf("%d", q.MessagesReady)),
			style.Width(10).Align(lipgloss.Right).Render(fmt.Sprintf("%d", q.MessagesUnacked)),
			style.Width(10).Align(lipgloss.Right).Render(fmt.Sprintf("%d", q.TotalMessages)),
			style.Width(10).Align(lipgloss.Right).Render(fmt.Sprintf("%d", q.Consumers)),
			style.Width(12).Render(status),
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Left, row...))
	}

	table := lipgloss.JoinVertical(lipgloss.Left, rows...)

	return BoxStyle.Render(table)
}

func (m MultiDashboardModel) getQueueStatus(q QueueData) string {
	if q.Consumers == 0 && q.TotalMessages > 0 {
		return lipgloss.NewStyle().Foreground(ErrorColor).Bold(true).Render("⚠ Sem consumers")
	}
	if q.TotalMessages > 1000 {
		return lipgloss.NewStyle().Foreground(WarningColor).Bold(true).Render("🔴 Alto volume")
	}
	if q.TotalMessages > 100 {
		return lipgloss.NewStyle().Foreground(WarningColor).Render("🟡 Moderado")
	}
	if q.TotalMessages > 0 {
		return lipgloss.NewStyle().Foreground(SuccessColor).Render("🟢 Normal")
	}
	return lipgloss.NewStyle().Foreground(MutedColor).Render("⚪ Vazia")
}

func (m MultiDashboardModel) renderSummary() string {
	if len(m.queuesData) == 0 {
		return ""
	}

	var totalReady, totalUnacked, totalMsgs, totalConsumers int
	var alertQueues, emptyQueues int

	for _, q := range m.queuesData {
		totalReady += q.MessagesReady
		totalUnacked += q.MessagesUnacked
		totalMsgs += q.TotalMessages
		totalConsumers += q.Consumers

		if q.Consumers == 0 && q.TotalMessages > 0 {
			alertQueues++
		}
		if q.TotalMessages == 0 {
			emptyQueues++
		}
	}

	// Estatísticas
	stats := []string{
		lipgloss.NewStyle().Foreground(AccentColor).Bold(true).Render("📈 Resumo"),
		"",
		fmt.Sprintf("  Total Ready:     %s", lipgloss.NewStyle().Foreground(SuccessColor).Bold(true).Render(fmt.Sprintf("%d", totalReady))),
		fmt.Sprintf("  Total Unacked:   %s", lipgloss.NewStyle().Foreground(WarningColor).Bold(true).Render(fmt.Sprintf("%d", totalUnacked))),
		fmt.Sprintf("  Total Mensagens: %s", lipgloss.NewStyle().Foreground(InfoColor).Bold(true).Render(fmt.Sprintf("%d", totalMsgs))),
		fmt.Sprintf("  Total Consumers: %s", lipgloss.NewStyle().Foreground(CyanColor).Bold(true).Render(fmt.Sprintf("%d", totalConsumers))),
		"",
	}

	// Alertas
	if alertQueues > 0 {
		stats = append(stats, lipgloss.NewStyle().Foreground(ErrorColor).Bold(true).Render(
			fmt.Sprintf("  ⚠ %d fila(s) sem consumers!", alertQueues)))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, stats...)
	return BoxHighlightStyle.Width(40).Render(content)
}

func (m MultiDashboardModel) renderFooter() string {
	keys := []string{
		lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("↑↓") + " navegar",
		lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("r") + " atualizar",
		lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("q") + " sair",
	}

	footer := strings.Join(keys, "  •  ")
	return lipgloss.NewStyle().
		Foreground(MutedColor).
		Align(lipgloss.Center).
		Width(m.width).
		Render(footer)
}

// fetchAllData busca dados de todas as filas
func (m MultiDashboardModel) fetchAllData() tea.Msg {
	mgmtClient := rabbitmq.NewManagementClient(m.cfg.RabbitMQ)

	var queuesData []QueueData

	for _, queueName := range m.queueNames {
		vhost := m.cfg.RabbitMQ.VHost
		if vhost == "" {
			vhost = "/"
		}

		queueInfo, err := mgmtClient.GetQueue(vhost, queueName)
		if err != nil {
			// Se não encontrar a fila, adicionar com zeros
			queuesData = append(queuesData, QueueData{
				Name:  queueName,
				Type:  "unknown",
				VHost: vhost,
			})
			continue
		}

		queuesData = append(queuesData, QueueData{
			Name:            queueInfo.Name,
			MessagesReady:   queueInfo.MessagesReady,
			MessagesUnacked: queueInfo.MessagesUnacked,
			TotalMessages:   queueInfo.Messages,
			Consumers:       queueInfo.Consumers,
			Type:            queueInfo.Type,
			VHost:           vhost,
		})
	}

	return multiUpdateMsg{
		queuesData: queuesData,
		err:        nil,
	}
}

func multiTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return multiTickMsg{t}
	})
}

func multiTickAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return multiTickMsg{t}
	})
}

// RunMultiDashboard executa o dashboard de múltiplas filas
func RunMultiDashboard(queueNames []string, cfg *config.Config, interval time.Duration) error {
	model := NewMultiDashboard(queueNames, cfg, interval)
	p := tea.NewProgram(model, tea.WithAltScreen())

	_, err := p.Run()
	return err
}
