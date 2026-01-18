package commands

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/davioliveeira/gohop/internal/config"
	"github.com/davioliveeira/gohop/internal/rabbitmq"
	"github.com/davioliveeira/gohop/internal/ui"
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
	monitorCmd.Flags().Int("interval", 20, "Intervalo de atualização em segundos")
}

func runMonitor(cmd *cobra.Command, args []string) error {
	queueName := args[0]

	fmt.Print(ui.SubMenuHeader("📊", "Monitorar Fila", fmt.Sprintf("Dashboard em tempo real para '%s'", queueName)))

	cfg, err := config.Load(profile)
	if err != nil {
		fmt.Println(ui.SubMenuError("Erro ao carregar configuração"))
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	intervalSec, _ := cmd.Flags().GetInt("interval")
	interval := time.Duration(intervalSec) * time.Second

	if interval < 1*time.Second {
		interval = 1 * time.Second
	}

	// Verificar fila
	fmt.Println(ui.SubMenuLoading("Verificando fila"))

	mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
	vhost := cfg.RabbitMQ.VHost
	if vhost == "" {
		vhost = "/"
	} else if vhost[0] != '/' {
		vhost = "/" + vhost
	}

	queue, err := mgmtClient.GetQueue(vhost, queueName)
	if err != nil {
		fmt.Println(ui.SubMenuError("Fila não encontrada"))
		return fmt.Errorf("fila não encontrada: %s", queueName)
	}

	fmt.Println(ui.SubMenuDone("Fila encontrada"))
	fmt.Println()

	// Preview da fila
	fmt.Println(ui.SubMenuSection("📋", "Status Atual"))
	fmt.Print(ui.SubMenuKeyValue("Mensagens prontas:", fmt.Sprintf("%d", queue.MessagesReady), queue.MessagesReady > 100))
	fmt.Print(ui.SubMenuKeyValue("Mensagens unacked:", fmt.Sprintf("%d", queue.MessagesUnacked), queue.MessagesUnacked > 0))
	fmt.Print(ui.SubMenuKeyValue("Consumers:", fmt.Sprintf("%d", queue.Consumers), queue.Consumers == 0))
	fmt.Print(ui.SubMenuKeyValue("Intervalo:", fmt.Sprintf("%ds", intervalSec), false))
	fmt.Println()

	fmt.Println(ui.SubMenuInfo("Iniciando dashboard interativo..."))
	fmt.Println(ui.SubMenuHelp("Pressione 'q' para sair, 'r' para atualizar"))
	fmt.Println()

	// Pequena pausa para usuário ler
	time.Sleep(time.Second)

	// Criar e executar dashboard
	model := ui.NewDashboard(queueName, cfg, interval)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("erro ao executar dashboard: %w", err)
	}

	// Limpar tela ao sair
	fmt.Print("\033[2J\033[H")
	fmt.Println(ui.SubMenuDone("Dashboard encerrado"))

	return nil
}
