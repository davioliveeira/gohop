package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Paleta de cores consistente para toda a CLI
var (
	// Cores primárias
	PrimaryColor     = lipgloss.Color("205") // Magenta/Pink - cor principal
	SecondaryColor   = lipgloss.Color("63")  // Roxo claro
	AccentColor      = lipgloss.Color("213") // Magenta claro
	BackgroundColor  = lipgloss.Color("236") // Cinza escuro para backgrounds
	BorderColor      = lipgloss.Color("237") // Cinza para bordas

	// Cores semânticas
	SuccessColor     = lipgloss.Color("42")  // Verde
	ErrorColor       = lipgloss.Color("196") // Vermelho
	WarningColor     = lipgloss.Color("220") // Amarelo
	InfoColor        = lipgloss.Color("39")  // Azul claro
	CyanColor        = lipgloss.Color("51")  // Ciano
	MutedColor       = lipgloss.Color("240") // Cinza claro
	MutedColorDark   = lipgloss.Color("238") // Cinza mais escuro

	// Texto
	TextPrimary      = lipgloss.Color("255") // Branco
	TextSecondary    = lipgloss.Color("243") // Cinza claro
	TextMuted        = lipgloss.Color("240") // Cinza médio
)

// Estilos globais reutilizáveis
var (
	// Header Styles
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(TextPrimary).
			Background(PrimaryColor).
			Padding(0, 1).
			MarginBottom(1)

	HeaderBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder(), true).
				BorderForeground(PrimaryColor).
				Padding(0, 1).
				MarginBottom(1)

	SubHeaderStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true).
			Padding(0, 1).
			MarginBottom(1)

	// Box/Border Styles
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderColor).
			Padding(1, 2).
			MarginBottom(1)

	BoxHighlightStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(PrimaryColor).
				Padding(1, 2).
				MarginBottom(1)

	// Label/Value Styles
	LabelStyle = lipgloss.NewStyle().
			Foreground(AccentColor).
			Bold(true)

	ValueStyle = lipgloss.NewStyle().
			Foreground(SuccessColor)

	ValueDangerStyle = lipgloss.NewStyle().
				Foreground(ErrorColor).
				Bold(true)

	// Status Styles
	StatusSuccessStyle = lipgloss.NewStyle().
				Foreground(SuccessColor).
				Bold(true)

	StatusErrorStyle = lipgloss.NewStyle().
				Foreground(ErrorColor).
				Bold(true)

	StatusWarningStyle = lipgloss.NewStyle().
				Foreground(WarningColor).
				Bold(true)

	StatusInfoStyle = lipgloss.NewStyle().
			Foreground(InfoColor).
			Bold(true)

	// Help/Footer Styles
	HelpStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			Italic(true).
			MarginTop(1)

	// Separator
	SeparatorStyle = lipgloss.NewStyle().
			Foreground(MutedColorDark).
			Render("─")
)

// RenderHeader cria um cabeçalho estilizado e animado
func RenderHeader(title string, subtitle string) string {
	var header strings.Builder
	
	titleStyled := HeaderStyle.Render(title)
	header.WriteString(titleStyled)
	
	if subtitle != "" {
		subtitleStyled := SubHeaderStyle.Render(subtitle)
		header.WriteString("\n")
		header.WriteString(subtitleStyled)
	}
	
	return header.String()
}

// RenderBox cria uma caixa estilizada com conteúdo
func RenderBox(content string, highlight bool) string {
	if highlight {
		return BoxHighlightStyle.Render(content)
	}
	return BoxStyle.Render(content)
}

// RenderSeparator cria um separador horizontal
func RenderSeparator(width int) string {
	return lipgloss.NewStyle().
		Foreground(MutedColorDark).
		Render(strings.Repeat("─", width))
}
