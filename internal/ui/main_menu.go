package ui

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/davioliveeira/gohop/internal/config"
)

// ═══════════════════════════════════════════════════════════════════════════════
// CORES E ESTILOS DO MENU
// ═══════════════════════════════════════════════════════════════════════════════

var (
	// Cores vibrantes para o menu
	menuPink       = lipgloss.Color("#FF6B9D")
	menuPurple     = lipgloss.Color("#C678DD")
	menuBlue       = lipgloss.Color("#61AFEF")
	menuCyan       = lipgloss.Color("#56B6C2")
	menuGreen      = lipgloss.Color("#98C379")
	menuYellow     = lipgloss.Color("#E5C07B")
	menuOrange     = lipgloss.Color("#D19A66")
	menuRed        = lipgloss.Color("#E06C75")
	menuWhite      = lipgloss.Color("#ABB2BF")
	menuGray       = lipgloss.Color("#5C6370")
	menuDarkGray   = lipgloss.Color("#3E4451")
	menuBackground = lipgloss.Color("#282C34")

	// Estilo do logo/título
	logoStyle = lipgloss.NewStyle().
			Foreground(menuPink).
			Bold(true)

	// Estilo da borda principal
	mainBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(menuPurple).
			Padding(1, 2)

	// Estilo do item selecionado
	selectedItemStyle = lipgloss.NewStyle().
				Foreground(menuBackground).
				Background(menuPink).
				Bold(true).
				Padding(0, 2)

	// Estilo do item normal
	normalItemStyle = lipgloss.NewStyle().
			Foreground(menuWhite).
			Padding(0, 2)

	// Estilo da descrição do item selecionado
	selectedDescStyle = lipgloss.NewStyle().
				Foreground(menuPink).
				Italic(true).
				PaddingLeft(4)

	// Estilo da descrição normal
	normalDescStyle = lipgloss.NewStyle().
			Foreground(menuGray).
			Italic(true).
			PaddingLeft(4)

	// Estilo do footer/help
	footerStyle = lipgloss.NewStyle().
			Foreground(menuGray).
			MarginTop(1)

	// Estilo do separador
	separatorStyle = lipgloss.NewStyle().
			Foreground(menuDarkGray)

	// Estilo do subtítulo
	subtitleStyle = lipgloss.NewStyle().
			Foreground(menuCyan).
			Italic(true)
)

// ═══════════════════════════════════════════════════════════════════════════════
// MODELO DO MENU
// ═══════════════════════════════════════════════════════════════════════════════

type menuOption struct {
	icon        string
	title       string
	description string
	command     string
	color       lipgloss.Color
}

type mainMenuModel struct {
	options     []menuOption
	cursor      int
	cfg         *config.Config
	quitting    bool
	selectedCmd string
	width       int
	height      int
	
	// Animação
	animPhase   float64
	tickCount   int
}

// Mensagem de tick para animações
type menuTickMsg time.Time

// ═══════════════════════════════════════════════════════════════════════════════
// FUNÇÕES DO MENU
// ═══════════════════════════════════════════════════════════════════════════════

func executeCommand(args ...string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("erro ao obter caminho do executável: %w", err)
	}

	execPath, err = filepath.Abs(execPath)
	if err != nil {
		return fmt.Errorf("erro ao resolver caminho: %w", err)
	}

	cmd := exec.Command(execPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("erro ao executar comando: %w", err)
	}

	return nil
}

func NewMainMenu(cfg *config.Config) mainMenuModel {
	options := []menuOption{
		{
			icon:        "⚙",
			title:       "Configurar Conexão",
			description: "Configurar conexão com RabbitMQ",
			command:     "config init",
			color:       menuBlue,
		},
		{
			icon:        "📋",
			title:       "Listar Filas",
			description: "Ver todas as filas disponíveis",
			command:     "queue list",
			color:       menuCyan,
		},
		{
			icon:        "➕",
			title:       "Criar Fila",
			description: "Criar nova fila com retry/DLQ",
			command:     "queue create",
			color:       menuGreen,
		},
		{
			icon:        "🔧",
			title:       "Reconfigurar Fila",
			description: "Adicionar retry sem perder mensagens",
			command:     "queue reconfigure",
			color:       menuPurple,
		},
		{
			icon:        "🗑",
			title:       "Deletar Fila",
			description: "Remover fila e mensagens",
			command:     "queue delete",
			color:       menuRed,
		},
		{
			icon:        "🧹",
			title:       "Limpar Fila",
			description: "Remover todas as mensagens (purge)",
			command:     "queue purge",
			color:       menuYellow,
		},
		{
			icon:        "📊",
			title:       "Monitorar Fila",
			description: "Dashboard em tempo real",
			command:     "monitor",
			color:       menuOrange,
		},
		{
			icon:        "✕",
			title:       "Sair",
			description: "Fechar aplicação",
			command:     "quit",
			color:       menuRed,
		},
	}

	return mainMenuModel{
		options:   options,
		cursor:    0,
		cfg:       cfg,
		width:     80,
		height:    24,
		animPhase: 0,
	}
}

func (m mainMenuModel) Init() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		return menuTickMsg(t)
	})
}

func (m mainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case menuTickMsg:
		m.tickCount++
		m.animPhase = float64(m.tickCount) * 0.1
		return m, tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
			return menuTickMsg(t)
		})

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.options) - 1
			}

		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}

		case "enter", " ":
			selected := m.options[m.cursor]
			if selected.command == "quit" {
				m.quitting = true
				return m, tea.Quit
			}
			m.selectedCmd = selected.command
			m.quitting = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m mainMenuModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// ═══════════════════════════════════════════════════════════════════════════
	// LOGO ANIMADO
	// ═══════════════════════════════════════════════════════════════════════════
	
	logo := m.renderLogo()
	b.WriteString(logo)
	b.WriteString("\n")

	// ═══════════════════════════════════════════════════════════════════════════
	// SUBTÍTULO
	// ═══════════════════════════════════════════════════════════════════════════
	
	subtitle := subtitleStyle.Render("RabbitMQ Queue Manager ✨")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, subtitle))
	b.WriteString("\n\n")

	// ═══════════════════════════════════════════════════════════════════════════
	// MENU DE OPÇÕES
	// ═══════════════════════════════════════════════════════════════════════════
	
	menuContent := m.renderMenuItems()
	
	// Box com borda arredondada
	menuBox := mainBoxStyle.
		Width(50).
		Render(menuContent)
	
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, menuBox))
	b.WriteString("\n")

	// ═══════════════════════════════════════════════════════════════════════════
	// FOOTER COM ATALHOS
	// ═══════════════════════════════════════════════════════════════════════════
	
	footer := m.renderFooter()
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, footer))

	return b.String()
}

func (m mainMenuModel) renderLogo() string {
	// Logo ASCII art minimalista e elegante
	logo := `
   ╔═══════════════════════════════════════════════════════════╗
   ║                                                           ║
   ║     ██████╗  ██████╗ ██╗  ██╗ ██████╗ ██████╗             ║
   ║    ██╔════╝ ██╔═══██╗██║  ██║██╔═══██╗██╔══██╗            ║
   ║    ██║  ███╗██║   ██║███████║██║   ██║██████╔╝            ║
   ║    ██║   ██║██║   ██║██╔══██║██║   ██║██╔═══╝             ║
   ║    ╚██████╔╝╚██████╔╝██║  ██║╚██████╔╝██║        🐰       ║
   ║     ╚═════╝  ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚═╝                 ║
   ║                                                           ║
   ╚═══════════════════════════════════════════════════════════╝
`

	// Estilo elegante - magenta/rosa com brilho sutil
	mainColor := menuPink
	
	// Efeito de brilho pulsante sutil
	pulse := math.Sin(m.animPhase) * 0.3 + 0.7
	var style lipgloss.Style
	if pulse > 0.85 {
		style = lipgloss.NewStyle().Foreground(menuPurple).Bold(true)
	} else {
		style = lipgloss.NewStyle().Foreground(mainColor).Bold(true)
	}

	// Renderizar logo centralizado
	styledLogo := style.Render(logo)
	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, styledLogo)
}

func (m mainMenuModel) renderMenuItems() string {
	var items strings.Builder

	for i, opt := range m.options {
		isSelected := i == m.cursor

		// Indicador de seleção animado
		var indicator string
		if isSelected {
			// Animação pulsante para o indicador
			pulse := math.Sin(m.animPhase*2) * 0.5 + 0.5
			if pulse > 0.5 {
				indicator = "▸ "
			} else {
				indicator = "► "
			}
		} else {
			indicator = "  "
		}

		// Ícone com cor
		iconStyle := lipgloss.NewStyle().Foreground(opt.color)
		icon := iconStyle.Render(opt.icon)

		// Título
		var titleStr string
		if isSelected {
			titleStr = selectedItemStyle.Render(opt.title)
		} else {
			titleStr = normalItemStyle.Render(opt.title)
		}

		// Linha do item
		indicatorStyle := lipgloss.NewStyle().Foreground(menuPink).Bold(true)
		itemLine := indicatorStyle.Render(indicator) + icon + " " + titleStr
		items.WriteString(itemLine)
		items.WriteString("\n")

		// Descrição
		var descStr string
		if isSelected {
			descStr = selectedDescStyle.Render(opt.description)
		} else {
			descStr = normalDescStyle.Render(opt.description)
		}
		items.WriteString(descStr)
		
		// Separador entre itens (exceto no último)
		if i < len(m.options)-1 {
			items.WriteString("\n")
			separator := separatorStyle.Render(strings.Repeat("─", 46))
			items.WriteString(separator)
			items.WriteString("\n")
		}
	}

	return items.String()
}

func (m mainMenuModel) renderFooter() string {
	// Atalhos de teclado com estilo
	keyStyle := lipgloss.NewStyle().
		Foreground(menuPink).
		Bold(true)
	
	descStyle := lipgloss.NewStyle().
		Foreground(menuGray)

	sep := descStyle.Render(" • ")

	footer := keyStyle.Render("↑↓") + descStyle.Render(" navegar") + sep +
		keyStyle.Render("enter") + descStyle.Render(" selecionar") + sep +
		keyStyle.Render("q") + descStyle.Render(" sair")

	return footerStyle.Render(footer)
}

// ═══════════════════════════════════════════════════════════════════════════════
// EXECUÇÃO DO MENU
// ═══════════════════════════════════════════════════════════════════════════════

func RunMainMenu(cfg *config.Config) error {
	if !isTerminal() {
		return fmt.Errorf("terminal não interativo. Use 'gohop --help' para ver comandos")
	}

	// Loop principal - mantém o menu aberto até o usuário escolher sair
	for {
		model := NewMainMenu(cfg)
		p := tea.NewProgram(model, tea.WithAltScreen())

		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("erro ao executar menu: %w", err)
		}

		menuModel, ok := finalModel.(mainMenuModel)
		if !ok {
			return nil
		}

		// Se o usuário escolheu sair ou fechou o menu sem selecionar nada
		if menuModel.selectedCmd == "" || menuModel.selectedCmd == "quit" {
			fmt.Println("\n👋 Até logo!")
			return nil
		}

		// Executar comando selecionado
		err = executeSelectedCommand(cfg, menuModel.selectedCmd)
		if err != nil {
			fmt.Println()
			fmt.Println(RenderErrorMessage(err.Error()))
		}

		// Aguardar usuário pressionar Enter para voltar ao menu
		fmt.Println()
		pressEnterStyle := lipgloss.NewStyle().
			Foreground(menuGray).
			Italic(true)
		fmt.Println(pressEnterStyle.Render("Pressione ENTER para voltar ao menu..."))
		fmt.Scanln()
	}
}

// executeSelectedCommand executa o comando selecionado no menu
func executeSelectedCommand(cfg *config.Config, selectedCmd string) error {
	switch selectedCmd {
	case "queue create":
		result, err := RunQueueCreateForm(cfg)
		if err != nil {
			return fmt.Errorf("erro no formulário: %w", err)
		}

		if err := CreateQueueFromForm(cfg, result); err != nil {
			return fmt.Errorf("erro ao criar fila: %w", err)
		}

		fmt.Println()
		fmt.Println(RenderSuccessMessage(fmt.Sprintf("Fila '%s' criada com sucesso!", result.QueueName)))
		if result.WithRetry {
			fmt.Println(RenderInfoMessage("Sistema de retry/DLQ configurado automaticamente"))
		}
		return nil

	case "queue reconfigure":
		result, err := RunReconfigureQueueForm(cfg)
		if err != nil {
			return fmt.Errorf("erro no formulário: %w", err)
		}

		if err := ReconfigureQueueWithRetry(cfg, result); err != nil {
			return fmt.Errorf("erro ao reconfigurar fila: %w", err)
		}

		fmt.Println()
		fmt.Println(RenderSuccessMessage(fmt.Sprintf("Fila '%s' reconfigurada com sucesso!", result.QueueName)))
		return nil

	case "queue delete":
		if err := RunDeleteQueueForm(cfg); err != nil {
			return fmt.Errorf("erro ao deletar fila: %w", err)
		}
		return nil

	case "queue purge":
		if err := RunPurgeQueueForm(cfg); err != nil {
			return fmt.Errorf("erro ao limpar fila: %w", err)
		}
		return nil

	case "monitor":
		var monitorMode string
		modeForm := huh.NewForm(
			huh.NewGroup(
				huh.NewNote().
					Title("📊 Monitorar Filas").
					Description("Escolha o modo de monitoramento"),

				huh.NewSelect[string]().
					Title("Modo de Monitoramento").
					Description("Como deseja monitorar?").
					Options(
						huh.NewOption("📊 Uma Fila Específica", "single"),
						huh.NewOption("📈 Múltiplas Filas Simultâneas", "multiple"),
					).
					Value(&monitorMode),
			),
		)
		modeForm.WithTheme(huh.ThemeCharm())

		if err := modeForm.Run(); err != nil {
			return fmt.Errorf("erro ao escolher modo: %w", err)
		}

		if monitorMode == "single" {
			selectedQueue, err := RunSingleMonitorForm(cfg)
			if err != nil {
				return fmt.Errorf("erro ao selecionar fila: %w", err)
			}

			args := []string{"monitor", selectedQueue}
			return executeCommand(args...)
		} else {
			// Monitoramento de múltiplas filas
			selectedQueues, err := RunMonitorQueueForm(cfg)
			if err != nil {
				return fmt.Errorf("erro ao selecionar filas: %w", err)
			}

			if len(selectedQueues) == 0 {
				return fmt.Errorf("nenhuma fila selecionada")
			}

			// Executar dashboard de múltiplas filas
			return RunMultiDashboard(selectedQueues, cfg, 20*time.Second)
		}

	default:
		args := strings.Fields(selectedCmd)
		return executeCommand(args...)
	}
}

func isTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
