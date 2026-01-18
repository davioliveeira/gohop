package ui

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Logger global estilizado para a CLI
	Logger *log.Logger
)

// InitLogger inicializa o logger estilizado do Charm Log
func InitLogger() {
	Logger = log.NewWithOptions(os.Stdout, log.Options{
		Prefix: "🐰 gohop",
		TimeFormat: "15:04:05",
		Level: log.InfoLevel,
		ReportCaller: false,
		ReportTimestamp: true,
	})

	// Configurar estilos customizados
	styles := log.DefaultStyles()
	
	// Estilo para INFO
	styles.Levels[log.InfoLevel] = lipgloss.NewStyle().
		Foreground(InfoColor).
		Bold(true).
		Padding(0, 1)
	
	// Estilo para WARN
	styles.Levels[log.WarnLevel] = lipgloss.NewStyle().
		Foreground(WarningColor).
		Bold(true).
		Padding(0, 1)
	
	// Estilo para ERROR
	styles.Levels[log.ErrorLevel] = lipgloss.NewStyle().
		Foreground(ErrorColor).
		Bold(true).
		Background(lipgloss.Color("52")).
		Padding(0, 1)
	
	// Estilo para DEBUG
	styles.Levels[log.DebugLevel] = lipgloss.NewStyle().
		Foreground(MutedColor).
		Italic(true).
		Padding(0, 1)
	
	Logger.SetStyles(styles)
}

// LogSuccess registra uma mensagem de sucesso
func LogSuccess(message string, keyvals ...interface{}) {
	Logger.Info(message, keyvals...)
}

// LogError registra uma mensagem de erro
func LogError(message string, err error, keyvals ...interface{}) {
	if err != nil {
		keyvals = append(keyvals, "error", err.Error())
	}
	Logger.Error(message, keyvals...)
}

// LogWarning registra uma mensagem de aviso
func LogWarning(message string, keyvals ...interface{}) {
	Logger.Warn(message, keyvals...)
}

// LogInfo registra uma mensagem informativa
func LogInfo(message string, keyvals ...interface{}) {
	Logger.Info(message, keyvals...)
}

// LogDebug registra uma mensagem de debug
func LogDebug(message string, keyvals ...interface{}) {
	Logger.Debug(message, keyvals...)
}
