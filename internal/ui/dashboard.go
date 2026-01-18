package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/davioliveeira/rabbit/internal/config"
	"github.com/davioliveeira/rabbit/internal/rabbitmq"
	"github.com/davioliveeira/rabbit/internal/retry"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			Align(lipgloss.Center).
			Padding(1)

	boxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("213")).
			Bold(true)

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
)

// QueueData representa dados de uma fila para o dashboard
type QueueData struct {
	Name           string
	MessagesReady  int
	MessagesUnacked int
	TotalMessages  int
	Consumers      int
	Type           string
	VHost          string
}

// RetryData representa dados do sistema de retry
type RetryData struct {
	MainQueueExists bool
	WaitQueueExists bool
	DLQExists       bool
	MainQueueMsgs   int
	WaitQueueMsgs   int
	DLQMsgs         int
	MaxRetries      int
	RetryDelay      int
}

// Model representa o estado do dashboard
type Model struct {
	queueName     string
	cfg           *config.Config
	queueData     *QueueData
	retryData     *RetryData
	spinner       spinner.Model
	loading       bool
	error         string
	interval      time.Duration
	lastUpdate    time.Time
	width         int
	height        int
}

// tickMsg é enviado periodicamente para atualizar o dashboard
type tickMsg struct {
	t time.Time
}

// updateMsg contém dados atualizados
type updateMsg struct {
	queueData *QueueData
	retryData *RetryData
	err       error
}

// NewDashboard cria um novo modelo de dashboard
func NewDashboard(queueName string, cfg *config.Config, interval time.Duration) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		queueName: queueName,
		cfg:       cfg,
		spinner:   s,
		loading:   true,
		interval:  interval,
	}
}

// Init inicializa o modelo
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.fetchData,
		tick(),
	)
}

// Update processa mensagens e atualiza o modelo
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r", "R":
			// Atualizar manualmente
			m.loading = true
			return m, tea.Batch(
				m.spinner.Tick,
				m.fetchData,
			)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		m.loading = true
		return m, tea.Batch(
			m.spinner.Tick,
			m.fetchData,
			tickAfter(m.interval),
		)

	case updateMsg:
		m.loading = false
		m.lastUpdate = time.Now()
		if msg.err != nil {
			m.error = msg.err.Error()
		} else {
			m.error = ""
			m.queueData = msg.queueData
			m.retryData = msg.retryData
		}
		return m, tickAfter(m.interval)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, tea.Batch(cmds...)
}

// View renderiza a interface
func (m Model) View() string {
	if m.width == 0 {
		return "Carregando..."
	}

	var sections []string

	// Título
	title := titleStyle.Render(fmt.Sprintf("🐰 RabbitMQ Monitor - %s", m.queueName))
	sections = append(sections, title)

	// Status de loading
	if m.loading {
		sections = append(sections, fmt.Sprintf("%s Atualizando...", m.spinner.View()))
	} else if m.error != "" {
		sections = append(sections, warningStyle.Render(fmt.Sprintf("❌ Erro: %s", m.error)))
	}

	// Dados da fila principal
	if m.queueData != nil {
		queueBox := boxStyle.Render(m.renderQueueBox(*m.queueData))
		sections = append(sections, queueBox)
	}

	// Dados do sistema de retry
	if m.retryData != nil {
		retryBox := boxStyle.Render(m.renderRetryBox(*m.retryData))
		sections = append(sections, retryBox)
	}

	// Informações de atualização
	updateInfo := fmt.Sprintf("\nÚltima atualização: %s | Intervalo: %v", 
		m.lastUpdate.Format("15:04:05"), m.interval)
	sections = append(sections, helpStyle.Render(updateInfo))

	// Ajuda
	help := helpStyle.Render("\nTeclas: [q/ESC] Sair | [r] Atualizar | [Ctrl+C] Sair")
	sections = append(sections, help)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderQueueBox renderiza o box da fila principal
func (m Model) renderQueueBox(data QueueData) string {
	var lines []string

	lines = append(lines, labelStyle.Render("📊 Main Queue"))
	lines = append(lines, "")

	lines = append(lines, fmt.Sprintf("%s %s", 
		labelStyle.Render("Nome:"), data.Name))
	lines = append(lines, fmt.Sprintf("%s %s", 
		labelStyle.Render("Tipo:"), data.Type))
	lines = append(lines, fmt.Sprintf("%s %s", 
		labelStyle.Render("VHost:"), data.VHost))
	lines = append(lines, "")

	lines = append(lines, fmt.Sprintf("%s %s", 
		labelStyle.Render("Mensagens Prontas:"), 
		valueStyle.Render(fmt.Sprintf("%d", data.MessagesReady))))
	lines = append(lines, fmt.Sprintf("%s %s", 
		labelStyle.Render("Mensagens Unacked:"), 
		valueStyle.Render(fmt.Sprintf("%d", data.MessagesUnacked))))
	lines = append(lines, fmt.Sprintf("%s %s", 
		labelStyle.Render("Total de Mensagens:"), 
		valueStyle.Render(fmt.Sprintf("%d", data.TotalMessages))))
	lines = append(lines, fmt.Sprintf("%s %s", 
		labelStyle.Render("Consumers:"), 
		valueStyle.Render(fmt.Sprintf("%d", data.Consumers))))

	// Barra visual de mensagens (ASCII art)
	if data.TotalMessages > 0 {
		lines = append(lines, "")
		bar := m.renderMessageBar(data.TotalMessages, 20)
		lines = append(lines, bar)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderRetryBox renderiza o box do sistema de retry
func (m Model) renderRetryBox(data RetryData) string {
	var lines []string

	lines = append(lines, labelStyle.Render("🔄 Sistema de Retry"))
	lines = append(lines, "")

	// Status dos componentes
	statusIcon := func(exists bool) string {
		if exists {
			return valueStyle.Render("✅")
		}
		return warningStyle.Render("❌")
	}

	lines = append(lines, fmt.Sprintf("%s Main Queue:  %s %d msgs", 
		statusIcon(data.MainQueueExists),
		labelStyle.Render(""), data.MainQueueMsgs))
	lines = append(lines, fmt.Sprintf("%s Wait Queue:  %s %d msgs", 
		statusIcon(data.WaitQueueExists),
		labelStyle.Render(""), data.WaitQueueMsgs))
	lines = append(lines, fmt.Sprintf("%s DLQ:         %s %d msgs", 
		statusIcon(data.DLQExists),
		labelStyle.Render(""), data.DLQMsgs))
	lines = append(lines, "")

	// Configurações
	lines = append(lines, fmt.Sprintf("%s %d", 
		labelStyle.Render("Max Retries:"), data.MaxRetries))
	lines = append(lines, fmt.Sprintf("%s %ds", 
		labelStyle.Render("Retry Delay:"), data.RetryDelay))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderMessageBar renderiza uma barra visual de mensagens
func (m Model) renderMessageBar(messages, width int) string {
	if messages == 0 {
		return helpStyle.Render(strings.Repeat("░", width) + " 0")
	}

	// Calcular porcentagem (máx 100%)
	max := 1000 // considerar 1000 como máximo para visualização
	percentage := float64(messages) / float64(max) * 100
	if percentage > 100 {
		percentage = 100
	}

	filled := int(float64(width) * percentage / 100)
	empty := width - filled

	bar := valueStyle.Render(strings.Repeat("█", filled)) +
		helpStyle.Render(strings.Repeat("░", empty))

	return fmt.Sprintf("%s %d", bar, messages)
}

// fetchData busca dados atualizados
func (m Model) fetchData() tea.Msg {
	mgmtClient := rabbitmq.NewManagementClient(m.cfg.RabbitMQ)

	vhost := m.cfg.RabbitMQ.VHost
	if vhost == "" {
		vhost = "/"
	} else if vhost[0] != '/' {
		vhost = "/" + vhost
	}

	// Buscar dados da fila principal
	queue, err := mgmtClient.GetQueue(vhost, m.queueName)
	if err != nil {
		return updateMsg{
			queueData: nil,
			retryData: nil,
			err:       err,
		}
	}

	queueData := &QueueData{
		Name:           queue.Name,
		MessagesReady:  queue.MessagesReady,
		MessagesUnacked: queue.MessagesUnacked,
		TotalMessages:  queue.Messages,
		Consumers:      queue.Consumers,
		Type:           queue.Type,
		VHost:          queue.VHost,
	}

	// Buscar dados do sistema de retry
	retryInfo, err := retry.GetRetrySystemInfo(mgmtClient, m.cfg.RabbitMQ, m.queueName)
	var retryData *RetryData
	if err == nil {
		retryData = &RetryData{
			MainQueueExists: retryInfo.MainQueue,
			WaitQueueExists: retryInfo.WaitQueue,
			DLQExists:       retryInfo.DLQ,
			MainQueueMsgs:   retryInfo.MainQueueMsgs,
			WaitQueueMsgs:   retryInfo.WaitQueueMsgs,
			DLQMsgs:         retryInfo.DLQMsgs,
			MaxRetries:      retryInfo.MaxRetries,
			RetryDelay:      retryInfo.RetryDelay,
		}
	}

	return updateMsg{
		queueData: queueData,
		retryData: retryData,
		err:       nil,
	}
}

// tick envia mensagem de atualização periódica
func tick() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return tickMsg{t: t}
	})
}

// tickAfter envia mensagem após intervalo específico
func tickAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg{t: t}
	})
}
