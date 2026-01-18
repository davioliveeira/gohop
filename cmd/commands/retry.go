package commands

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/davioliveeira/rabbit/internal/config"
	"github.com/davioliveeira/rabbit/internal/rabbitmq"
	"github.com/davioliveeira/rabbit/internal/retry"
	"github.com/spf13/cobra"
)

var retryCmd = &cobra.Command{
	Use:   "retry",
	Short: "Gerenciar sistema de retry",
	Long:  "Comandos para configurar e gerenciar sistema de retry para filas",
}

var retrySetupCmd = &cobra.Command{
	Use:   "setup [queue-name]",
	Short: "Configurar sistema de retry para uma fila",
	Long: `Cria o sistema completo de retry para uma fila:

Arquitetura:
1. Main Queue -> (reject) -> Wait Exchange -> Wait Queue (TTL) -> Retry Exchange
2. Retry Exchange -> routes back to Main Queue (se retries < MAX_RETRIES)
3. Retry Exchange -> routes to DLQ (se retries >= MAX_RETRIES)`,
	Args: cobra.ExactArgs(1),
	RunE: runRetrySetup,
}

var retryStatusCmd = &cobra.Command{
	Use:   "status [queue-name]",
	Short: "Status do sistema de retry",
	Long:  "Mostra informações sobre o sistema de retry de uma fila",
	Args:  cobra.ExactArgs(1),
	RunE:  runRetryStatus,
}

func init() {
	retrySetupCmd.Flags().Int("max-retries", 3, "Número máximo de tentativas")
	retrySetupCmd.Flags().Int("retry-delay", 5, "Delay entre tentativas (segundos)")
	retrySetupCmd.Flags().Int("dlq-ttl", 604800000, "TTL de mensagens na DLQ (milissegundos)")
	retrySetupCmd.Flags().Bool("force", false, "Recriar mesmo se já existir")
	retrySetupCmd.Flags().String("queue-type", "quorum", "Tipo da fila principal (classic|quorum)")

	retryCmd.AddCommand(retrySetupCmd)
	retryCmd.AddCommand(retryStatusCmd)
}

func runRetrySetup(cmd *cobra.Command, args []string) error {
	queueName := args[0]

	// Carregar configuração
	cfg, err := config.Load(profile)
	if err != nil {
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	// Obter flags
	maxRetries, _ := cmd.Flags().GetInt("max-retries")
	retryDelay, _ := cmd.Flags().GetInt("retry-delay")
	dlqTTL, _ := cmd.Flags().GetInt("dlq-ttl")
	force, _ := cmd.Flags().GetBool("force")
	queueType, _ := cmd.Flags().GetString("queue-type")

	// Mostrar preview do que será criado
	fmt.Println()
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	fmt.Println(titleStyle.Render(fmt.Sprintf("🔧 Configurando Retry para: %s", queueName)))
	fmt.Println()

	waitQueueName := fmt.Sprintf("%s.wait", queueName)
	waitExchangeName := fmt.Sprintf("%s.wait.exchange", queueName)
	retryExchangeName := fmt.Sprintf("%s.retry", queueName)
	dlqName := fmt.Sprintf("%s.dlq", queueName)

	fmt.Println("📋 Componentes do Sistema de Retry:")
	fmt.Printf("  📦 Main Queue:       %s\n", queueName)
	fmt.Printf("  ⏱️  Wait Queue:       %s (TTL: %ds)\n", waitQueueName, retryDelay)
	fmt.Printf("  🔄 Wait Exchange:    %s\n", waitExchangeName)
	fmt.Printf("  🔀 Retry Exchange:   %s (headers)\n", retryExchangeName)
	fmt.Printf("  💀 DLQ:              %s (TTL: %dms)\n", dlqName, dlqTTL)
	fmt.Printf("  🔢 Max Retries:      %d\n", maxRetries)
	fmt.Println()

	// Conectar
	client, err := rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		return fmt.Errorf("erro ao conectar: %w", err)
	}
	defer client.Close()

	// Verificar se fila principal existe
	mainQueueExists, err := client.QueueExists(queueName)
	if err != nil {
		return fmt.Errorf("erro ao verificar fila: %w", err)
	}

	if mainQueueExists && !force {
		// Perguntar se quer recriar a fila principal
		var recreate bool
		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Fila '%s' já existe", queueName)).
					Description("Para configurar retry, é necessário recriar a fila com DLX. Deseja continuar?").
					Value(&recreate),
			),
		)
		confirmForm.WithTheme(huh.ThemeCharm())

		if err := confirmForm.Run(); err != nil {
			return err
		}

		if !recreate {
			fmt.Println("❌ Operação cancelada")
			return nil
		}

		// Deletar fila principal
		fmt.Printf("🗑️  Deletando fila '%s'...\n", queueName)
		_, err := client.DeleteQueue(queueName, false, false, false)
		if err != nil {
			// Tentar via Management API
			mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
			vhost := cfg.RabbitMQ.VHost
			if vhost == "" {
				vhost = "/"
			} else if vhost[0] != '/' {
				vhost = "/" + vhost
			}
			if err := mgmtClient.DeleteQueueViaAPI(vhost, queueName); err != nil {
				return fmt.Errorf("erro ao deletar fila: %w", err)
			}
		}
		fmt.Printf("✅ Fila '%s' deletada\n", queueName)
	}

	// Configurar sistema de retry
	setupOpts := retry.SetupOptions{
		QueueName:  queueName,
		MaxRetries: maxRetries,
		RetryDelay: retryDelay,
		DLQTTL:     dlqTTL,
		Force:      force,
	}

	fmt.Println("⚙️  Configurando sistema de retry...")
	if err := retry.SetupRetry(client, setupOpts); err != nil {
		return fmt.Errorf("erro ao configurar retry: %w", err)
	}

	fmt.Println("  ✅ Wait exchange criado")
	fmt.Println("  ✅ Wait queue criada")
	fmt.Println("  ✅ Retry exchange criado")
	fmt.Println("  ✅ DLQ criada")
	fmt.Println("  ✅ Bindings configurados")

	// Recriar fila principal com DLX
	if !mainQueueExists || force {
		fmt.Println()
		fmt.Printf("📦 Recriando fila '%s' com DLX...\n", queueName)
		if err := retry.RecreateQueueWithDLX(client, queueName, queueType); err != nil {
			return fmt.Errorf("erro ao recriar fila principal: %w", err)
		}
		fmt.Printf("  ✅ Fila '%s' recriada com DLX apontando para %s\n", queueName, waitExchangeName)
	}

	// Mensagem final
	fmt.Println()
	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)

	fmt.Println(successStyle.Render("✅ Sistema de retry configurado com sucesso!"))
	fmt.Println()
	fmt.Println("ℹ️  Próximos passos:")
	fmt.Println("   1. Configure seu consumer para verificar header 'x-death'")
	fmt.Println("   2. Se retry_count < MAX_RETRIES, publique novamente na main queue")
	fmt.Println("   3. Se retry_count >= MAX_RETRIES, publique na DLQ ou aceite o erro")

	return nil
}

func runRetryStatus(cmd *cobra.Command, args []string) error {
	queueName := args[0]

	// Carregar configuração
	cfg, err := config.Load(profile)
	if err != nil {
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	// Obter informações do sistema de retry
	mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
	retryInfo, err := retry.GetRetrySystemInfo(mgmtClient, cfg.RabbitMQ, queueName)
	if err != nil {
		return fmt.Errorf("erro ao obter informações: %w", err)
	}

	// Formatar saída
	fmt.Println()
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		PaddingBottom(1)

	fmt.Println(titleStyle.Render(fmt.Sprintf("📊 Status do Sistema de Retry: %s", queueName)))
	fmt.Println()

	// Status dos componentes
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	missingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	fmt.Println("🔧 Componentes:")
	fmt.Printf("  📦 Main Queue:     ")
	if retryInfo.MainQueue {
		fmt.Println(statusStyle.Render("✅") + fmt.Sprintf(" (%d msgs)", retryInfo.MainQueueMsgs))
	} else {
		fmt.Println(missingStyle.Render("❌ Não encontrada"))
	}

	fmt.Printf("  ⏱️  Wait Queue:     ")
	if retryInfo.WaitQueue {
		fmt.Println(statusStyle.Render("✅") + fmt.Sprintf(" (%d msgs)", retryInfo.WaitQueueMsgs))
	} else {
		fmt.Println(missingStyle.Render("❌ Não encontrada"))
	}

	fmt.Printf("  🔄 Wait Exchange:  ")
	if retryInfo.WaitExchange {
		fmt.Println(statusStyle.Render("✅"))
	} else {
		fmt.Println(missingStyle.Render("❌ Não encontrada"))
	}

	fmt.Printf("  🔀 Retry Exchange: ")
	if retryInfo.RetryExchange {
		fmt.Println(statusStyle.Render("✅"))
	} else {
		fmt.Println(missingStyle.Render("❌ Não encontrada"))
	}

	fmt.Printf("  💀 DLQ:            ")
	if retryInfo.DLQ {
		fmt.Println(statusStyle.Render("✅") + fmt.Sprintf(" (%d msgs)", retryInfo.DLQMsgs))
	} else {
		fmt.Println(missingStyle.Render("❌ Não encontrada"))
	}

	fmt.Println()

	// Configurações
	fmt.Println("⚙️  Configurações:")
	fmt.Printf("  Max Retries:  %d\n", retryInfo.MaxRetries)
	fmt.Printf("  Retry Delay:  %ds\n", retryInfo.RetryDelay)
	fmt.Printf("  DLQ TTL:      %dms\n", retryInfo.DLQTTL)

	// Verificar se está completo
	fmt.Println()
	if retryInfo.MainQueue && retryInfo.WaitQueue && retryInfo.WaitExchange && retryInfo.RetryExchange && retryInfo.DLQ {
		completeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)
		fmt.Println(completeStyle.Render("✅ Sistema de retry completo e funcionando"))
	} else {
		incompleteStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
		fmt.Println(incompleteStyle.Render("⚠️  Sistema de retry incompleto. Execute 'retry setup' para configurar."))
	}

	return nil
}
