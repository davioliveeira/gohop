package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/davioliveeira/gohop/internal/config"
	"github.com/davioliveeira/gohop/internal/rabbitmq"
	"github.com/davioliveeira/gohop/internal/retry"
)

// Estilos do dashboard - usando tema unificado do theme.go
// Criamos aliases locais para manter compatibilidade com código existente
var (
	// titleStyle: estilo de título para dashboard
	titleStyle = SubHeaderStyle.Copy().Align(lipgloss.Center)
	
	// boxStyle: alias para BoxStyle do theme.go
	boxStyle = BoxStyle
	
	// labelStyle: alias para LabelStyle do theme.go
	labelStyle = LabelStyle
	
	// valueStyle: alias para ValueStyle do theme.go
	valueStyle = ValueStyle
	
	// warningStyle: alias para StatusWarningStyle do theme.go
	warningStyle = StatusWarningStyle
	
	// helpStyle: alias para HelpStyle do theme.go
	helpStyle = HelpStyle
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
	
	// Animações Harmonica
	headerAnim    HeaderAnimationModel
	progressAnim  ProgressAnimationModel
	cycle         float64 // Para animações de ciclo (pulsing)
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
	s.Style = lipgloss.NewStyle().Foreground(PrimaryColor) // Usando tema unificado

	// Inicializar animações Harmonica
	headerAnim := NewHeaderAnimationModel()
	progressAnim := NewProgressAnimationModel()

	return Model{
		queueName:    queueName,
		cfg:          cfg,
		spinner:      s,
		loading:      true,
		interval:     interval,
		headerAnim:   headerAnim,
		progressAnim: progressAnim,
		cycle:        0.0,
	}
}

// Init inicializa o modelo
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.fetchData,
		tick(),
		m.headerAnim.Init(),
		// Animação de ciclo para efeito pulsante
		tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
			return cycleMsg{}
		}),
	)
}

// cycleMsg atualiza o ciclo para animações pulsantes
type cycleMsg struct{}

// progressUpdateMsg atualiza a animação de progresso
type progressUpdateMsg struct {
	Delta float64
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
			
			// Atualizar animação de progresso baseada nos dados
			if msg.queueData != nil && msg.queueData.TotalMessages > 0 {
				// Normalizar progresso para 0.0-1.0 (usando 1000 como máximo)
				maxMsgs := float64(1000)
				progress := float64(msg.queueData.TotalMessages) / maxMsgs
				if progress > 1.0 {
					progress = 1.0
				}
				m.progressAnim.SetProgress(progress)
				m.progressAnim.width = 18
			}
		}
		return m, tea.Batch(
			tickAfter(m.interval),
			// Continuar animações
			tea.Tick(16*time.Millisecond, func(t time.Time) tea.Msg {
				return progressUpdateMsg{Delta: float64(16 * time.Millisecond) / float64(time.Second)}
			}),
		)

	case cycleMsg:
		// Atualizar ciclo para animações pulsantes (0.0 a 1.0)
		m.cycle += 0.05
		if m.cycle > 1.0 {
			m.cycle = 0.0
		}
		return m, tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
			return cycleMsg{}
		})

	case HeaderAnimationMsg:
		// Atualizar animação de header
		updatedHeader, cmd := m.headerAnim.Update(msg)
		m.headerAnim = updatedHeader
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case progressUpdateMsg:
		// Atualizar animação de progresso
		m.progressAnim.Update(msg.Delta)
		// Continuar atualizando se ainda estiver animando
		if m.progressAnim.animation.IsAnimating() {
			cmds = append(cmds, tea.Tick(16*time.Millisecond, func(t time.Time) tea.Msg {
				return progressUpdateMsg{Delta: msg.Delta}
			}))
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renderiza a interface
func (m Model) View() string {
	if m.width == 0 {
		return "Carregando..."
	}

	var b strings.Builder

	// ═══════════════════════════════════════════════════════════════════════
	// HEADER
	// ═══════════════════════════════════════════════════════════════════════
	header := m.renderHeader()
	b.WriteString(header)
	b.WriteString("\n\n")

	// ═══════════════════════════════════════════════════════════════════════
	// CONTEÚDO PRINCIPAL
	// ═══════════════════════════════════════════════════════════════════════
	
	if m.loading && m.queueData == nil {
		// Loading inicial
		loadingBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(2, 4).
			Width(60).
			Align(lipgloss.Center).
			Render(fmt.Sprintf("%s Carregando dados da fila...", m.spinner.View()))
		b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, loadingBox))
	} else if m.error != "" {
		// Erro
		errorBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ErrorColor).
			Padding(1, 2).
			Width(60).
			Render(lipgloss.NewStyle().Foreground(ErrorColor).Render(fmt.Sprintf("❌ %s", m.error)))
		b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, errorBox))
	} else {
		// Layout em duas colunas
		content := m.renderContent()
		b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, content))
	}

	b.WriteString("\n\n")

	// ═══════════════════════════════════════════════════════════════════════
	// FOOTER
	// ═══════════════════════════════════════════════════════════════════════
	footer := m.renderFooter()
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, footer))

	return b.String()
}

func (m Model) renderHeader() string {
	// Logo/Título animado
	logoStyle := lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true)

	logo := `
    ____       _     _     _ _   __  __  ___  
   |  _ \ __ _| |__ | |__ (_) |_|  \/  |/ _ \ 
   | |_) / _` + "`" + ` | '_ \| '_ \| | __| |\/| | | | |
   |  _ < (_| | |_) | |_) | | |_| |  | | |_| |
   |_| \_\__,_|_.__/|_.__/|_|\__|_|  |_|\__\_\
`
	styledLogo := logoStyle.Render(logo)

	// Subtítulo com nome da fila
	subtitleStyle := lipgloss.NewStyle().
		Foreground(AccentColor).
		Bold(true)
	subtitle := subtitleStyle.Render(fmt.Sprintf("📊 Monitorando: %s", m.queueName))

	// Status de atualização
	var statusLine string
	if m.loading {
		statusLine = lipgloss.NewStyle().
			Foreground(WarningColor).
			Render(fmt.Sprintf("%s Atualizando...", m.spinner.View()))
	} else {
		statusLine = lipgloss.NewStyle().
			Foreground(SuccessColor).
			Render(fmt.Sprintf("✓ Atualizado às %s", m.lastUpdate.Format("15:04:05")))
	}

	header := lipgloss.JoinVertical(lipgloss.Center, styledLogo, subtitle, statusLine)
	return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(header)
}

func (m Model) renderContent() string {
	// Box da fila principal
	queueBox := m.renderQueueBoxNew()
	
	// Box do sistema de retry (se existir)
	var retryBox string
	if m.retryData != nil {
		retryBox = m.renderRetryBoxNew()
	}

	// Layout lado a lado
	if retryBox != "" {
		return lipgloss.JoinHorizontal(lipgloss.Top, queueBox, "  ", retryBox)
	}
	return queueBox
}

func (m Model) renderQueueBoxNew() string {
	if m.queueData == nil {
		return ""
	}

	data := *m.queueData

	// Estilos
	titleStyle := lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(MutedColor).
		Width(18)

	valueStyle := lipgloss.NewStyle().
		Foreground(TextPrimary).
		Bold(true)

	// Construir conteúdo
	var lines []string

	// Título
	lines = append(lines, titleStyle.Render("📨 FILA PRINCIPAL"))
	lines = append(lines, "")

	// Informações básicas
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("Nome"),
		valueStyle.Render(data.Name)))

	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("Tipo"),
		lipgloss.NewStyle().Foreground(InfoColor).Render(data.Type)))

	// Separador
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(MutedColorDark).Render(strings.Repeat("─", 35)))
	lines = append(lines, "")

	// Métricas com cores
	readyColor := SuccessColor
	if data.MessagesReady > 100 {
		readyColor = WarningColor
	}
	if data.MessagesReady > 1000 {
		readyColor = ErrorColor
	}

	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("📥 Ready"),
		lipgloss.NewStyle().Foreground(readyColor).Bold(true).Render(fmt.Sprintf("%d", data.MessagesReady))))

	unackedColor := MutedColor
	if data.MessagesUnacked > 0 {
		unackedColor = WarningColor
	}
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("⏳ Unacked"),
		lipgloss.NewStyle().Foreground(unackedColor).Bold(true).Render(fmt.Sprintf("%d", data.MessagesUnacked))))

	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("📊 Total"),
		lipgloss.NewStyle().Foreground(InfoColor).Bold(true).Render(fmt.Sprintf("%d", data.TotalMessages))))

	// Separador
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(MutedColorDark).Render(strings.Repeat("─", 35)))
	lines = append(lines, "")

	// Consumers
	consumerColor := ErrorColor
	consumerIcon := "⚠"
	if data.Consumers > 0 {
		consumerColor = SuccessColor
		consumerIcon = "👥"
	}
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render(fmt.Sprintf("%s Consumers", consumerIcon)),
		lipgloss.NewStyle().Foreground(consumerColor).Bold(true).Render(fmt.Sprintf("%d", data.Consumers))))

	// Barra de progresso visual
	if data.TotalMessages > 0 {
		lines = append(lines, "")
		lines = append(lines, m.renderVisualBar(data.TotalMessages, 1000, 30))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	// Box com borda
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Padding(1, 2).
		Width(42).
		Render(content)
}

func (m Model) renderRetryBoxNew() string {
	if m.retryData == nil {
		return ""
	}

	data := *m.retryData

	// Estilos
	titleStyle := lipgloss.NewStyle().
		Foreground(SecondaryColor).
		Bold(true).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(MutedColor).
		Width(16)

	// Construir conteúdo
	var lines []string

	// Título
	lines = append(lines, titleStyle.Render("🔄 SISTEMA DE RETRY"))
	lines = append(lines, "")

	// Status dos componentes
	checkIcon := func(exists bool) string {
		if exists {
			return lipgloss.NewStyle().Foreground(SuccessColor).Render("✓")
		}
		return lipgloss.NewStyle().Foreground(ErrorColor).Render("✗")
	}

	// Main Queue
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		checkIcon(data.MainQueueExists),
		lipgloss.NewStyle().Foreground(TextPrimary).Render(" Main Queue"),
	))
	if data.MainQueueMsgs > 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(MutedColor).PaddingLeft(2).Render(
			fmt.Sprintf("└─ %d msgs", data.MainQueueMsgs)))
	}

	lines = append(lines, "")

	// Wait Queue
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		checkIcon(data.WaitQueueExists),
		lipgloss.NewStyle().Foreground(TextPrimary).Render(" Wait Queue"),
	))
	waitColor := MutedColor
	if data.WaitQueueMsgs > 0 {
		waitColor = WarningColor
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(waitColor).PaddingLeft(2).Render(
		fmt.Sprintf("└─ %d msgs aguardando retry", data.WaitQueueMsgs)))

	lines = append(lines, "")

	// DLQ
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		checkIcon(data.DLQExists),
		lipgloss.NewStyle().Foreground(TextPrimary).Render(" Dead Letter Queue"),
	))
	dlqColor := SuccessColor
	dlqIcon := "└─"
	if data.DLQMsgs > 0 {
		dlqColor = ErrorColor
		dlqIcon = "└─ ⚠"
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(dlqColor).PaddingLeft(2).Render(
		fmt.Sprintf("%s %d msgs falharam", dlqIcon, data.DLQMsgs)))

	// Separador
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(MutedColorDark).Render(strings.Repeat("─", 30)))
	lines = append(lines, "")

	// Configuração
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("Max Retries"),
		lipgloss.NewStyle().Foreground(InfoColor).Bold(true).Render(fmt.Sprintf("%d", data.MaxRetries))))

	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("Retry Delay"),
		lipgloss.NewStyle().Foreground(InfoColor).Bold(true).Render(fmt.Sprintf("%ds", data.RetryDelay))))

	// Status geral
	lines = append(lines, "")
	if data.DLQMsgs > 10 {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true).
			Render("⚠ ATENÇÃO: Muitas falhas na DLQ!"))
	} else if data.WaitQueueMsgs > 0 {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(WarningColor).
			Render("⏳ Mensagens aguardando retry..."))
	} else {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(SuccessColor).
			Render("✓ Sistema funcionando normalmente"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	// Box com borda
	borderColor := SecondaryColor
	if data.DLQMsgs > 10 {
		borderColor = ErrorColor
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width(38).
		Render(content)
}

func (m Model) renderVisualBar(current, max, width int) string {
	if max == 0 {
		max = 1
	}

	percent := float64(current) / float64(max)
	if percent > 1.0 {
		percent = 1.0
	}

	filled := int(float64(width) * percent)
	empty := width - filled

	// Cor baseada no percentual
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

	percentStr := fmt.Sprintf(" %.0f%%", percent*100)
	if current > max {
		percentStr = fmt.Sprintf(" %d+", current)
	}

	return filledBar + emptyBar + lipgloss.NewStyle().Foreground(MutedColor).Render(percentStr)
}

func (m Model) renderFooter() string {
	// Teclas de atalho
	keyStyle := lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(MutedColor)
	sepStyle := lipgloss.NewStyle().Foreground(MutedColorDark)

	keys := []string{
		keyStyle.Render("r") + descStyle.Render(" atualizar"),
		keyStyle.Render("q") + descStyle.Render(" sair"),
	}

	keysLine := strings.Join(keys, sepStyle.Render("  │  "))

	// Intervalo de atualização
	intervalLine := descStyle.Render(fmt.Sprintf("Atualização automática a cada %v", m.interval))

	footer := lipgloss.JoinVertical(lipgloss.Center, keysLine, intervalLine)

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(MutedColorDark).
		Padding(1, 0).
		Width(80).
		Align(lipgloss.Center).
		Render(footer)
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

	// Status de saúde da fila
	healthStatus := m.getQueueHealth(data)
	lines = append(lines, fmt.Sprintf("%s %s", 
		labelStyle.Render("Status:"), healthStatus))
	lines = append(lines, "")

	lines = append(lines, fmt.Sprintf("%s %s", 
		labelStyle.Render("Mensagens Prontas:"), 
		m.formatMessageCount(data.MessagesReady)))
	lines = append(lines, fmt.Sprintf("%s %s", 
		labelStyle.Render("Mensagens Unacked:"), 
		m.formatMessageCount(data.MessagesUnacked)))
	lines = append(lines, fmt.Sprintf("%s %s", 
		labelStyle.Render("Total de Mensagens:"), 
		m.formatMessageCount(data.TotalMessages)))
	lines = append(lines, fmt.Sprintf("%s %s", 
		labelStyle.Render("Consumers:"), 
		m.formatConsumerCount(data.Consumers)))

	lines = append(lines, "")

	// Barras de progresso para cada tipo de mensagem (com animação Harmonica para Total)
	lines = append(lines, labelStyle.Render("📈 Volume de Mensagens:"))
	lines = append(lines, "")
	
	// Usar animação Harmonica para progresso principal (Total)
	if data.TotalMessages > 0 {
		// Atualizar largura da animação
		m.progressAnim.width = 18
		// Calcular progresso normalizado
		maxMsgs := float64(1000)
		progress := float64(data.TotalMessages) / maxMsgs
		if progress > 1.0 {
			progress = 1.0
		}
		// Atualizar animação (se necessário)
		if math.Abs(m.progressAnim.target-progress) > 0.01 {
			m.progressAnim.SetProgress(progress)
		}
		lines = append(lines, fmt.Sprintf("  Total:     %s", m.progressAnim.View()))
	} else {
		lines = append(lines, fmt.Sprintf("  Total:     %s", 
			m.renderProgressBar(0, 18)))
	}
	
	// Outras barras sem animação (ou com animação simplificada)
	if data.MessagesReady > 0 {
		lines = append(lines, fmt.Sprintf("  Prontas:   %s", 
			m.renderProgressBar(data.MessagesReady, 18)))
	}
	if data.MessagesUnacked > 0 {
		lines = append(lines, fmt.Sprintf("  Unacked:   %s", 
			m.renderProgressBar(data.MessagesUnacked, 18)))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderRetryBox renderiza o box do sistema de retry
func (m Model) renderRetryBox(data RetryData) string {
	var lines []string

	lines = append(lines, labelStyle.Render("🔄 Sistema de Retry"))
	lines = append(lines, "")

	// Status geral do sistema de retry
	retryHealth := m.getRetrySystemHealth(data)
	lines = append(lines, fmt.Sprintf("%s %s", 
		labelStyle.Render("Status:"), retryHealth))
	lines = append(lines, "")

	// Status dos componentes com indicadores visuais
	statusIcon := func(exists bool) string {
		if exists {
			return valueStyle.Render("✅")
		}
		return warningStyle.Render("❌")
	}

	lines = append(lines, fmt.Sprintf("%s Main Queue:  %d msgs", 
		statusIcon(data.MainQueueExists), data.MainQueueMsgs))
	if data.MainQueueMsgs > 0 {
		lines = append(lines, fmt.Sprintf("  %s", 
			m.renderProgressBar(data.MainQueueMsgs, 16)))
	}

	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("%s Wait Queue:  %d msgs", 
		statusIcon(data.WaitQueueExists), data.WaitQueueMsgs))
	if data.WaitQueueMsgs > 0 {
		lines = append(lines, fmt.Sprintf("  %s", 
			m.renderProgressBar(data.WaitQueueMsgs, 16)))
	}

	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("%s DLQ:         %d msgs", 
		statusIcon(data.DLQExists), data.DLQMsgs))
	if data.DLQMsgs > 0 {
		// DLQ com mais mensagens indica problemas - usar cor vermelha
		dlqBar := m.renderProgressBar(data.DLQMsgs, 16)
		if data.DLQMsgs > 10 {
			dlqBar = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(dlqBar)
		}
		lines = append(lines, fmt.Sprintf("  %s", dlqBar))
	}

	lines = append(lines, "")

	// Configurações
	lines = append(lines, labelStyle.Render("⚙️  Configurações:"))
	lines = append(lines, fmt.Sprintf("  Max Retries:  %d", data.MaxRetries))
	lines = append(lines, fmt.Sprintf("  Retry Delay:  %ds", data.RetryDelay))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderMessageBar renderiza uma barra visual de mensagens melhorada
func (m Model) renderMessageBar(messages, width int) string {
	if messages == 0 {
		emptyBar := strings.Repeat("░", width)
		return helpStyle.Render(fmt.Sprintf("%s %d", emptyBar, messages))
	}

	// Calcular porcentagem (escala adaptativa baseada em threshold)
	max := 1000 // considerar 1000 como máximo para visualização
	percentage := float64(messages) / float64(max) * 100
	if percentage > 100 {
		percentage = 100
	}

	filled := int(float64(width) * percentage / 100)
	empty := width - filled

	// Cores adaptativas baseadas no volume
	var barStyle lipgloss.Style
	var emptyStyle = helpStyle
	if percentage < 30 {
		// Verde: volume baixo
		barStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	} else if percentage < 70 {
		// Amarelo: volume médio
		barStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	} else {
		// Vermelho: volume alto
		barStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	}

	bar := barStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", empty))

	return fmt.Sprintf("%s %d (%.1f%%)", bar, messages, percentage)
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
	return tea.Tick(time.Second*20, func(t time.Time) tea.Msg {
		return tickMsg{t: t}
	})
}

// tickAfter envia mensagem após intervalo específico
func tickAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg{t: t}
	})
}

// getQueueHealth retorna o status de saúde da fila com cor
func (m Model) getQueueHealth(data QueueData) string {
	// Verificar diferentes condições de saúde
	var status string
	var style lipgloss.Style

	if data.Consumers == 0 {
		status = "⚠️  Sem Consumers"
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	} else if data.MessagesUnacked > data.MessagesReady {
		status = "⚠️  Alta carga (Unacked > Ready)"
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	} else if data.TotalMessages > 1000 {
		status = "🔴 Volume Alto"
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	} else if data.TotalMessages > 100 {
		status = "🟡 Volume Médio"
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	} else {
		status = "✅ Saudável"
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	}

	return style.Render(status)
}

// getRetrySystemHealth retorna o status de saúde do sistema de retry
func (m Model) getRetrySystemHealth(data RetryData) string {
	if !data.MainQueueExists || !data.WaitQueueExists || !data.DLQExists {
		status := "❌ Incompleto"
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render(status)
	}

	if data.DLQMsgs > 10 {
		status := fmt.Sprintf("⚠️  DLQ com %d mensagens", data.DLQMsgs)
		return lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true).Render(status)
	}

	status := "✅ Configurado"
	return lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true).Render(status)
}

// renderProgressBar renderiza uma barra de progresso simples e colorida
func (m Model) renderProgressBar(current, width int) string {
	if current == 0 {
		return helpStyle.Render(strings.Repeat("░", width) + " 0")
	}

	// Escala adaptativa
	max := 1000
	percentage := float64(current) / float64(max) * 100
	if percentage > 100 {
		percentage = 100
	}

	filled := int(float64(width) * percentage / 100)
	empty := width - filled

	// Cores adaptativas
	var barStyle lipgloss.Style
	if percentage < 30 {
		barStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	} else if percentage < 70 {
		barStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	} else {
		barStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	}

	bar := barStyle.Render(strings.Repeat("█", filled)) +
		helpStyle.Render(strings.Repeat("░", empty))

	return fmt.Sprintf("%s %d", bar, current)
}

// formatMessageCount formata contagem de mensagens com cores adaptativas
func (m Model) formatMessageCount(count int) string {
	var style lipgloss.Style
	if count == 0 {
		style = helpStyle
	} else if count > 1000 {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	} else if count > 100 {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	} else {
		style = valueStyle
	}
	return style.Render(fmt.Sprintf("%d", count))
}

// formatConsumerCount formata contagem de consumers com cores adaptativas
func (m Model) formatConsumerCount(count int) string {
	var style lipgloss.Style
	if count == 0 {
		style = warningStyle
	} else if count >= 5 {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	} else {
		style = valueStyle
	}
	return style.Render(fmt.Sprintf("%d", count))
}
