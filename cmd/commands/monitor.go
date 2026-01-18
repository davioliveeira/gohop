package commands

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/davioliveeira/rabbit/internal/config"
	"github.com/davioliveeira/rabbit/internal/rabbitmq"
	"github.com/davioliveeira/rabbit/internal/ui"
	"github.com/spf13/cobra"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor [queue-name]",
	Short: "Monitorar filas em tempo real",
	Long: `Dashboard TUI interativo para monitorar status de filas em tempo real.

Mostra:
- Status da fila principal (mensagens, consumers)
- Status do sistema de retry (wait queue, DLQ)
- Atualização automática configurável

Teclas:
  q, ESC, Ctrl+C  - Sair
  r               - Atualizar manualmente`,
	Args: cobra.ExactArgs(1),
	RunE: runMonitor,
}

func init() {
	monitorCmd.Flags().Int("interval", 2, "Intervalo de atualização em segundos")
}

func runMonitor(cmd *cobra.Command, args []string) error {
	queueName := args[0]

	// Carregar configuração
	cfg, err := config.Load(profile)
	if err != nil {
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	// Obter intervalo
	intervalSec, _ := cmd.Flags().GetInt("interval")
	interval := time.Duration(intervalSec) * time.Second

	if interval < 1*time.Second {
		interval = 1 * time.Second
	}

	// Verificar se a fila existe antes de iniciar o dashboard
	mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
	vhost := cfg.RabbitMQ.VHost
	if vhost == "" {
		vhost = "/"
	} else if vhost[0] != '/' {
		vhost = "/" + vhost
	}

	_, err = mgmtClient.GetQueue(vhost, queueName)
	if err != nil {
		return fmt.Errorf("fila não encontrada: %s", queueName)
	}

	// Criar modelo do dashboard
	model := ui.NewDashboard(queueName, cfg, interval)

	// Criar programa Bubble Tea
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Executar programa
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("erro ao executar dashboard: %w", err)
	}

	// Limpar tela ao sair
	fmt.Print("\033[2J\033[H")

	return nil
}
