package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// ═══════════════════════════════════════════════════════════════════════════════
// ESTILOS PARA SUBMENUS
// ═══════════════════════════════════════════════════════════════════════════════

var (
	// Cores do submenu
	subPink     = lipgloss.Color("#FF6B9D")
	subPurple   = lipgloss.Color("#C678DD")
	subCyan     = lipgloss.Color("#56B6C2")
	subGreen    = lipgloss.Color("#98C379")
	subYellow   = lipgloss.Color("#E5C07B")
	subRed      = lipgloss.Color("#E06C75")
	subGray     = lipgloss.Color("#5C6370")
	subDarkGray = lipgloss.Color("#3E4451")
	subWhite    = lipgloss.Color("#ABB2BF")

	// Estilo do header do submenu
	subHeaderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subPurple).
			Padding(1, 3).
			MarginBottom(1)

	// Estilo do título
	subTitleStyle = lipgloss.NewStyle().
			Foreground(subPink).
			Bold(true)

	// Estilo do subtítulo
	subSubtitleStyle = lipgloss.NewStyle().
				Foreground(subGray).
				Italic(true)

	// Estilo para seções
	subSectionStyle = lipgloss.NewStyle().
			Foreground(subCyan).
			Bold(true).
			MarginTop(1)

	// Estilo para labels
	subLabelStyle = lipgloss.NewStyle().
			Foreground(subGray)

	// Estilo para valores
	subValueStyle = lipgloss.NewStyle().
			Foreground(subWhite)

	// Estilo para valores destacados
	subValueHighlightStyle = lipgloss.NewStyle().
				Foreground(subGreen).
				Bold(true)

	// Box para conteúdo
	subBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subDarkGray).
			Padding(1, 2).
			MarginBottom(1)

	// Separador
	subSeparatorStyle = lipgloss.NewStyle().
				Foreground(subDarkGray)
)

// ═══════════════════════════════════════════════════════════════════════════════
// COMPONENTES DE SUBMENU
// ═══════════════════════════════════════════════════════════════════════════════

// SubMenuHeader renderiza um header bonito para submenus
func SubMenuHeader(icon, title, subtitle string) string {
	var content strings.Builder

	// Título com ícone
	titleLine := subTitleStyle.Render(fmt.Sprintf("%s  %s", icon, title))
	content.WriteString(titleLine)

	// Subtítulo se fornecido
	if subtitle != "" {
		content.WriteString("\n")
		content.WriteString(subSubtitleStyle.Render(subtitle))
	}

	return subHeaderStyle.Render(content.String()) + "\n"
}

// SubMenuSection renderiza uma seção com título
func SubMenuSection(icon, title string) string {
	return subSectionStyle.Render(fmt.Sprintf("\n%s %s", icon, title)) + "\n"
}

// SubMenuSeparator renderiza um separador
func SubMenuSeparator(width int) string {
	return subSeparatorStyle.Render(strings.Repeat("─", width)) + "\n"
}

// SubMenuKeyValue renderiza um par chave-valor
func SubMenuKeyValue(key, value string, highlight bool) string {
	keyStr := subLabelStyle.Render(fmt.Sprintf("  %-20s", key))
	var valStr string
	if highlight {
		valStr = subValueHighlightStyle.Render(value)
	} else {
		valStr = subValueStyle.Render(value)
	}
	return keyStr + valStr + "\n"
}

// SubMenuBox renderiza conteúdo em um box
func SubMenuBox(content string) string {
	return subBoxStyle.Render(content)
}

// SubMenuStatus renderiza um status colorido
func SubMenuStatus(status string, statusType string) string {
	var icon string
	var style lipgloss.Style

	switch statusType {
	case "success":
		icon = "✓"
		style = lipgloss.NewStyle().Foreground(subGreen).Bold(true)
	case "error":
		icon = "✗"
		style = lipgloss.NewStyle().Foreground(subRed).Bold(true)
	case "warning":
		icon = "⚠"
		style = lipgloss.NewStyle().Foreground(subYellow).Bold(true)
	case "info":
		icon = "ℹ"
		style = lipgloss.NewStyle().Foreground(subCyan).Bold(true)
	default:
		icon = "•"
		style = lipgloss.NewStyle().Foreground(subWhite)
	}

	return style.Render(fmt.Sprintf("%s %s", icon, status))
}

// SubMenuList renderiza uma lista de itens
func SubMenuList(items []string, bullet string) string {
	var b strings.Builder
	for _, item := range items {
		b.WriteString(subLabelStyle.Render(fmt.Sprintf("  %s ", bullet)))
		b.WriteString(subValueStyle.Render(item))
		b.WriteString("\n")
	}
	return b.String()
}

// SubMenuTable renderiza uma tabela simples
func SubMenuTable(headers []string, rows [][]string) string {
	var b strings.Builder

	// Calcular larguras das colunas
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Header
	headerStyle := lipgloss.NewStyle().Foreground(subCyan).Bold(true)
	for i, h := range headers {
		b.WriteString(headerStyle.Render(fmt.Sprintf("%-*s  ", widths[i], h)))
	}
	b.WriteString("\n")

	// Separador
	for i := range headers {
		b.WriteString(subSeparatorStyle.Render(strings.Repeat("─", widths[i])))
		b.WriteString("  ")
	}
	b.WriteString("\n")

	// Rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				b.WriteString(subValueStyle.Render(fmt.Sprintf("%-*s  ", widths[i], cell)))
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

// ═══════════════════════════════════════════════════════════════════════════════
// TEMA CUSTOMIZADO PARA HUH FORMS
// ═══════════════════════════════════════════════════════════════════════════════

// GetCharmTheme retorna o tema customizado para formulários huh
func GetCharmTheme() *huh.Theme {
	t := huh.ThemeCharm()

	// Customizar estilos focados
	t.Focused.Title = t.Focused.Title.Foreground(subPink).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(subGray).Italic(true)
	t.Focused.Base = t.Focused.Base.BorderForeground(subPurple)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(subPink)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(subCyan)
	t.Focused.Option = t.Focused.Option.Foreground(subWhite)
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(subPink)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(subDarkGray)

	// Customizar estilos desfocados
	t.Blurred.Title = t.Blurred.Title.Foreground(subGray)
	t.Blurred.Description = t.Blurred.Description.Foreground(subDarkGray)

	return t
}

// ═══════════════════════════════════════════════════════════════════════════════
// HELPERS DE LOADING E PROGRESS
// ═══════════════════════════════════════════════════════════════════════════════

// SubMenuLoading renderiza uma mensagem de loading
func SubMenuLoading(message string) string {
	style := lipgloss.NewStyle().Foreground(subCyan)
	return style.Render(fmt.Sprintf("⏳ %s...", message))
}

// SubMenuDone renderiza uma mensagem de conclusão
func SubMenuDone(message string) string {
	style := lipgloss.NewStyle().Foreground(subGreen).Bold(true)
	return style.Render(fmt.Sprintf("✓ %s", message))
}

// SubMenuError renderiza uma mensagem de erro
func SubMenuError(message string) string {
	style := lipgloss.NewStyle().Foreground(subRed).Bold(true)
	return style.Render(fmt.Sprintf("✗ %s", message))
}

// SubMenuWarning renderiza uma mensagem de aviso
func SubMenuWarning(message string) string {
	style := lipgloss.NewStyle().Foreground(subYellow).Bold(true)
	return style.Render(fmt.Sprintf("⚠ %s", message))
}

// SubMenuInfo renderiza uma mensagem informativa
func SubMenuInfo(message string) string {
	style := lipgloss.NewStyle().Foreground(subCyan)
	return style.Render(fmt.Sprintf("ℹ %s", message))
}

// ═══════════════════════════════════════════════════════════════════════════════
// FOOTER E HELP
// ═══════════════════════════════════════════════════════════════════════════════

// SubMenuFooter renderiza um footer com atalhos
func SubMenuFooter(shortcuts map[string]string) string {
	keyStyle := lipgloss.NewStyle().Foreground(subPink).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(subGray)
	sepStyle := lipgloss.NewStyle().Foreground(subDarkGray)

	var parts []string
	for key, desc := range shortcuts {
		parts = append(parts, keyStyle.Render(key)+descStyle.Render(" "+desc))
	}

	return "\n" + strings.Join(parts, sepStyle.Render(" • "))
}

// SubMenuHelp renderiza uma mensagem de ajuda
func SubMenuHelp(message string) string {
	style := lipgloss.NewStyle().Foreground(subGray).Italic(true)
	return "\n" + style.Render(message)
}
