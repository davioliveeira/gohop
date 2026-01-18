package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Estilos de progresso (usando paleta de cores do theme.go)
	// Estes estilos agora usam as cores definidas em theme.go para consistência
	progressTitleStyle = SubHeaderStyle.Copy().Padding(1, 0)

	progressStatusStyle = lipgloss.NewStyle().
		Foreground(MutedColor).
		Italic(true)

	progressSuccessStyle = StatusSuccessStyle

	progressErrorStyle = StatusErrorStyle

	progressWarningStyle = StatusWarningStyle
)

// ProgressTask representa uma tarefa no progresso
type ProgressTask struct {
	Name    string
	Status  string // "pending", "running", "done", "error"
	Message string
}

// progressModel representa o modelo de progresso
type progressModel struct {
	progress progress.Model
	tasks    []ProgressTask
	current  int
	total    int
	quitting bool
	done     bool
	width    int
}

// progressMsg atualiza o progresso
type progressMsg struct {
	TaskIndex int
	Status    string
	Message   string
}

// NewProgressModel cria um novo modelo de progresso
func NewProgressModel(total int, tasks []ProgressTask) progressModel {
	p := progress.New(progress.WithWidth(60), progress.WithDefaultGradient())

	return progressModel{
		progress: p,
		tasks:    tasks,
		total:    total,
		current:  0,
	}
}

func (m progressModel) Init() tea.Cmd {
	return m.progress.Init()
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.progress.Width = msg.Width - 4
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}

	case progressMsg:
		if msg.TaskIndex >= 0 && msg.TaskIndex < len(m.tasks) {
			m.tasks[msg.TaskIndex].Status = msg.Status
			m.tasks[msg.TaskIndex].Message = msg.Message

			// Atualizar contagem de tarefas concluídas
			doneCount := 0
			for _, task := range m.tasks {
				if task.Status == "done" {
					doneCount++
				} else if task.Status == "error" {
					doneCount++ // Contar erros como "concluídos" para progresso
				}
			}
			m.current = doneCount

			// Se todas as tarefas foram concluídas, marcar como done
			if m.current >= m.total {
				m.done = true
				return m, tea.Sequence(tea.Tick(time.Second, func(time.Time) tea.Msg {
					return tea.Quit
				}))
			}
		}
		return m, nil

	default:
		var cmd tea.Cmd
		var newProgress tea.Model
		newProgress, cmd = m.progress.Update(msg)
		if updated, ok := newProgress.(progress.Model); ok {
			m.progress = updated
		}
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m progressModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Calcular percentual
	percent := float64(m.current) / float64(m.total)
	m.progress.SetPercent(percent)

	// Título
	b.WriteString(progressTitleStyle.Render("⚙️  Processando..."))
	b.WriteString("\n\n")

	// Barra de progresso
	b.WriteString(m.progress.View())
	b.WriteString("\n\n")

	// Status: X/Y tarefas concluídas
	statusText := fmt.Sprintf("%d/%d tarefas concluídas", m.current, m.total)
	b.WriteString(progressStatusStyle.Render(statusText))
	b.WriteString("\n\n")

	// Lista de tarefas
	for i, task := range m.tasks {
		var icon string
		var style lipgloss.Style

		switch task.Status {
		case "done":
			icon = "✅"
			style = progressSuccessStyle
		case "error":
			icon = "❌"
			style = progressErrorStyle
		case "running":
			icon = "🔄"
			style = progressWarningStyle
		default: // pending
			icon = "⏳"
			style = progressStatusStyle
		}

		taskLine := fmt.Sprintf("%s %s", icon, task.Name)
		if task.Message != "" {
			taskLine += fmt.Sprintf(" - %s", task.Message)
		}

		b.WriteString(style.Render(taskLine))
		if i < len(m.tasks)-1 {
			b.WriteString("\n")
		}
	}

	// Help
	if !m.done {
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Pressione 'q' para cancelar (operações em andamento continuarão)"))
	}

	return b.String()
}

// SimpleProgressBar renderiza uma barra de progresso simples sem TUI
// Útil para comandos que não usam Bubble Tea
func SimpleProgressBar(current, total int, width int) string {
	if total == 0 {
		return ""
	}

	percent := float64(current) / float64(total)
	filled := int(float64(width) * percent)
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	percentStr := fmt.Sprintf("%.1f%%", percent*100)

	return fmt.Sprintf("[%s] %s (%d/%d)", bar, percentStr, current, total)
}

// RenderSuccessMessage renderiza uma mensagem de sucesso estilizada
func RenderSuccessMessage(message string) string {
	return progressSuccessStyle.Render(fmt.Sprintf("✅ %s", message))
}

// RenderErrorMessage renderiza uma mensagem de erro estilizada
func RenderErrorMessage(message string) string {
	return progressErrorStyle.Render(fmt.Sprintf("❌ %s", message))
}

// RenderWarningMessage renderiza uma mensagem de aviso estilizada
func RenderWarningMessage(message string) string {
	return progressWarningStyle.Render(fmt.Sprintf("⚠️  %s", message))
}

// RenderInfoMessage renderiza uma mensagem informativa estilizada
func RenderInfoMessage(message string) string {
	return progressStatusStyle.Render(fmt.Sprintf("ℹ️  %s", message))
}
