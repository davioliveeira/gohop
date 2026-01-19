package commands

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/davioliveeira/gohop/internal/config"
	"github.com/davioliveeira/gohop/internal/rabbitmq"
	"github.com/davioliveeira/gohop/internal/retry"
	"github.com/davioliveeira/gohop/internal/ui"
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

	fmt.Print(ui.SubMenuHeader("🔄", "Configurar Retry", fmt.Sprintf("Setup do sistema de retry para '%s'", queueName)))

	cfg, err := config.Load(profile)
	if err != nil {
		fmt.Println(ui.SubMenuError("Erro ao carregar configuração"))
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	maxRetries, _ := cmd.Flags().GetInt("max-retries")
	retryDelay, _ := cmd.Flags().GetInt("retry-delay")
	dlqTTL, _ := cmd.Flags().GetInt("dlq-ttl")
	force, _ := cmd.Flags().GetBool("force")
	queueType, _ := cmd.Flags().GetString("queue-type")

	// Mostrar componentes que serão criados
	fmt.Println(ui.SubMenuSection("📦", "Componentes do Sistema de Retry"))

	waitQueueName := fmt.Sprintf("%s.wait", queueName)
	waitExchangeName := fmt.Sprintf("%s.wait.exchange", queueName)
	retryExchangeName := fmt.Sprintf("%s.retry", queueName)
	dlqName := fmt.Sprintf("%s.dlq", queueName)

	fmt.Print(ui.SubMenuKeyValue("Main Queue:", queueName, true))
	fmt.Print(ui.SubMenuKeyValue("Wait Queue:", fmt.Sprintf("%s (TTL: %ds)", waitQueueName, retryDelay), false))
	fmt.Print(ui.SubMenuKeyValue("Wait Exchange:", waitExchangeName, false))
	fmt.Print(ui.SubMenuKeyValue("Retry Exchange:", fmt.Sprintf("%s (headers)", retryExchangeName), false))
	fmt.Print(ui.SubMenuKeyValue("DLQ:", fmt.Sprintf("%s (TTL: %dms)", dlqName, dlqTTL), false))
	fmt.Print(ui.SubMenuKeyValue("Max Retries:", strconv.Itoa(maxRetries), false))
	fmt.Println()

	// Conectar
	fmt.Println(ui.SubMenuLoading("Conectando ao RabbitMQ"))
	client, err := rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		fmt.Println(ui.SubMenuError("Erro ao conectar"))
		return fmt.Errorf("erro ao conectar: %w", err)
	}
	defer client.Close()

	// Verificar fila principal
	mainQueueExists, err := client.QueueExists(queueName)
	if err != nil {
		return fmt.Errorf("erro ao verificar fila: %w", err)
	}

	if mainQueueExists && !force {
		fmt.Println(ui.SubMenuWarning(fmt.Sprintf("Fila '%s' já existe", queueName)))
		fmt.Println()
		fmt.Println(ui.SubMenuInfo("Para configurar retry, é necessário recriar a fila com DLX"))
		fmt.Println()

		var recreate bool
		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("🗑️  Recriar fila?").
					Description("A fila atual será deletada e recriada com Dead Letter Exchange").
					Value(&recreate),
			),
		)
		confirmForm.WithTheme(ui.GetCharmTheme())

		if err := confirmForm.Run(); err != nil {
			return err
		}

		if !recreate {
			fmt.Println(ui.SubMenuError("Operação cancelada"))
			return nil
		}

		fmt.Println(ui.SubMenuLoading("Deletando fila existente"))
		_, err := client.DeleteQueue(queueName, false, false, false)
		if err != nil {
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
		fmt.Println(ui.SubMenuDone("Fila deletada"))
	}

	// Configurar sistema de retry
	fmt.Println()
	fmt.Println(ui.SubMenuSection("⚙", "Criando Componentes"))

	setupOpts := retry.SetupOptions{
		QueueName:  queueName,
		QueueType:  queueType,
		MaxRetries: maxRetries,
		RetryDelay: retryDelay,
		DLQTTL:     dlqTTL,
		Force:      force,
	}

	fmt.Println(ui.SubMenuLoading("Criando wait exchange"))
	if err := retry.SetupRetry(client, setupOpts); err != nil {
		fmt.Println(ui.SubMenuError("Erro ao configurar retry"))
		return fmt.Errorf("erro ao configurar retry: %w", err)
	}

	fmt.Println(ui.SubMenuDone("Wait exchange criado"))
	fmt.Println(ui.SubMenuDone("Wait queue criada"))
	fmt.Println(ui.SubMenuDone("Retry exchange criado"))
	fmt.Println(ui.SubMenuDone("DLQ criada"))
	fmt.Println(ui.SubMenuDone("Bindings configurados"))

	// Recriar fila principal
	if !mainQueueExists || force {
		fmt.Println()
		fmt.Println(ui.SubMenuLoading(fmt.Sprintf("Recriando fila '%s' com DLX", queueName)))
		if err := retry.RecreateQueueWithDLX(client, queueName, queueType); err != nil {
			fmt.Println(ui.SubMenuError("Erro ao recriar fila"))
			return fmt.Errorf("erro ao recriar fila: %w", err)
		}
		fmt.Println(ui.SubMenuDone(fmt.Sprintf("Fila '%s' recriada com DLX", queueName)))
	}

	// Sucesso
	fmt.Println()
	fmt.Println(ui.SubMenuDone("Sistema de retry configurado com sucesso!"))
	fmt.Println()

	fmt.Println(ui.SubMenuSection("📝", "Próximos Passos"))
	fmt.Print(ui.SubMenuList([]string{
		"Configure seu consumer para verificar header 'x-death'",
		"Se retry_count < MAX_RETRIES, republique na main queue",
		"Se retry_count >= MAX_RETRIES, trate como erro definitivo",
	}, "•"))

	return nil
}

func runRetryStatus(cmd *cobra.Command, args []string) error {
	queueName := args[0]

	fmt.Print(ui.SubMenuHeader("📊", "Status do Retry", fmt.Sprintf("Sistema de retry para '%s'", queueName)))

	cfg, err := config.Load(profile)
	if err != nil {
		fmt.Println(ui.SubMenuError("Erro ao carregar configuração"))
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	fmt.Println(ui.SubMenuLoading("Buscando informações"))

	mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
	retryInfo, err := retry.GetRetrySystemInfo(mgmtClient, cfg.RabbitMQ, queueName)
	if err != nil {
		fmt.Println(ui.SubMenuError("Erro ao obter informações"))
		return fmt.Errorf("erro ao obter informações: %w", err)
	}

	// Status dos componentes
	fmt.Println(ui.SubMenuSection("🔧", "Componentes"))

	// Main Queue
	if retryInfo.MainQueue {
		fmt.Print(ui.SubMenuKeyValue("Main Queue:", ui.SubMenuStatus(fmt.Sprintf("OK (%d msgs)", retryInfo.MainQueueMsgs), "success"), false))
	} else {
		fmt.Print(ui.SubMenuKeyValue("Main Queue:", ui.SubMenuStatus("Não encontrada", "error"), false))
	}

	// Wait Queue
	if retryInfo.WaitQueue {
		fmt.Print(ui.SubMenuKeyValue("Wait Queue:", ui.SubMenuStatus(fmt.Sprintf("OK (%d msgs)", retryInfo.WaitQueueMsgs), "success"), false))
	} else {
		fmt.Print(ui.SubMenuKeyValue("Wait Queue:", ui.SubMenuStatus("Não encontrada", "error"), false))
	}

	// Wait Exchange
	if retryInfo.WaitExchange {
		fmt.Print(ui.SubMenuKeyValue("Wait Exchange:", ui.SubMenuStatus("OK", "success"), false))
	} else {
		fmt.Print(ui.SubMenuKeyValue("Wait Exchange:", ui.SubMenuStatus("Não encontrado", "error"), false))
	}

	// Retry Exchange
	if retryInfo.RetryExchange {
		fmt.Print(ui.SubMenuKeyValue("Retry Exchange:", ui.SubMenuStatus("OK", "success"), false))
	} else {
		fmt.Print(ui.SubMenuKeyValue("Retry Exchange:", ui.SubMenuStatus("Não encontrado", "error"), false))
	}

	// DLQ
	if retryInfo.DLQ {
		dlqStatus := "success"
		if retryInfo.DLQMsgs > 0 {
			dlqStatus = "warning"
		}
		if retryInfo.DLQMsgs > 100 {
			dlqStatus = "error"
		}
		fmt.Print(ui.SubMenuKeyValue("DLQ:", ui.SubMenuStatus(fmt.Sprintf("OK (%d msgs)", retryInfo.DLQMsgs), dlqStatus), false))
	} else {
		fmt.Print(ui.SubMenuKeyValue("DLQ:", ui.SubMenuStatus("Não encontrada", "error"), false))
	}

	// Configurações
	fmt.Println(ui.SubMenuSection("⚙", "Configurações"))
	fmt.Print(ui.SubMenuKeyValue("Max Retries:", strconv.Itoa(retryInfo.MaxRetries), false))
	fmt.Print(ui.SubMenuKeyValue("Retry Delay:", fmt.Sprintf("%ds", retryInfo.RetryDelay), false))
	fmt.Print(ui.SubMenuKeyValue("DLQ TTL:", fmt.Sprintf("%dms", retryInfo.DLQTTL), false))

	// Resumo
	fmt.Println()
	if retryInfo.MainQueue && retryInfo.WaitQueue && retryInfo.WaitExchange && retryInfo.RetryExchange && retryInfo.DLQ {
		fmt.Println(ui.SubMenuDone("Sistema de retry completo e funcionando"))
	} else {
		fmt.Println(ui.SubMenuWarning("Sistema de retry incompleto"))
		fmt.Println(ui.SubMenuHelp("Execute 'gohop retry setup' para configurar"))
	}

	return nil
}
