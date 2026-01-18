package commands

import (
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/davioliveeira/gohop/internal/config"
	"github.com/davioliveeira/gohop/internal/rabbitmq"
	"github.com/davioliveeira/gohop/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Gerenciar filas RabbitMQ",
	Long:  "Comandos para criar, listar, deletar e gerenciar filas",
}

var queueCreateCmd = &cobra.Command{
	Use:   "create [nome]",
	Short: "Criar uma nova fila",
	Long:  "Cria uma nova fila com opções configuráveis",
	Args:  cobra.ExactArgs(1),
	RunE:  runQueueCreate,
}

var queueListCmd = &cobra.Command{
	Use:   "list",
	Short: "Listar todas as filas",
	Long:  "Lista todas as filas disponíveis no RabbitMQ",
	RunE:  runQueueList,
}

var queueDeleteCmd = &cobra.Command{
	Use:   "delete [nome]",
	Short: "Deletar uma fila",
	Long:  "Remove uma fila do RabbitMQ",
	Args:  cobra.ExactArgs(1),
	RunE:  runQueueDelete,
}

var queueStatusCmd = &cobra.Command{
	Use:   "status [nome]",
	Short: "Status de uma fila",
	Long:  "Mostra informações detalhadas sobre uma fila",
	Args:  cobra.ExactArgs(1),
	RunE:  runQueueStatus,
}

var queuePurgeCmd = &cobra.Command{
	Use:   "purge [nome]",
	Short: "Limpar todas as mensagens de uma fila",
	Long:  "Remove todas as mensagens prontas de uma fila sem deletá-la",
	Args:  cobra.ExactArgs(1),
	RunE:  runQueuePurge,
}

func init() {
	queueCreateCmd.Flags().String("type", "quorum", "Tipo de fila (classic|quorum)")
	queueCreateCmd.Flags().Bool("durable", true, "Fila durável")
	queueCreateCmd.Flags().Bool("auto-delete", false, "Auto-deletar quando não usada")
	queueCreateCmd.Flags().Bool("with-retry", false, "Configurar sistema de retry automaticamente")
	queueCreateCmd.Flags().Int("max-retries", 3, "Número máximo de tentativas")
	queueCreateCmd.Flags().Int("retry-delay", 5, "Delay entre tentativas (segundos)")
	queueCreateCmd.Flags().Bool("dry-run", false, "Mostrar o que seria criado sem executar")

	queueDeleteCmd.Flags().Bool("if-unused", false, "Só deletar se não tiver consumers")
	queueDeleteCmd.Flags().Bool("if-empty", false, "Só deletar se estiver vazia")
	queueDeleteCmd.Flags().Bool("cascade", false, "Deletar também filas relacionadas (wait, DLQ)")

	queueCmd.AddCommand(queueCreateCmd)
	queueCmd.AddCommand(queueListCmd)
	queueCmd.AddCommand(queueDeleteCmd)
	queueCmd.AddCommand(queueStatusCmd)
	queueCmd.AddCommand(queuePurgeCmd)
}

func runQueueCreate(cmd *cobra.Command, args []string) error {
	queueName := args[0]
	fmt.Print(ui.SubMenuHeader("➕", "Criar Fila", fmt.Sprintf("Criando fila '%s'", queueName)))

	cfg, err := config.Load(profile)
	if err != nil {
		fmt.Println(ui.SubMenuError("Erro ao carregar configuração"))
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	queueType, _ := cmd.Flags().GetString("type")
	durable, _ := cmd.Flags().GetBool("durable")
	autoDelete, _ := cmd.Flags().GetBool("auto-delete")
	withRetry, _ := cmd.Flags().GetBool("with-retry")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if dryRun {
		fmt.Println(ui.SubMenuSection("📋", "Dry Run - O que seria criado"))
		fmt.Print(ui.SubMenuKeyValue("Nome:", queueName, true))
		fmt.Print(ui.SubMenuKeyValue("Tipo:", queueType, false))
		fmt.Print(ui.SubMenuKeyValue("Durável:", fmt.Sprintf("%v", durable), false))
		fmt.Print(ui.SubMenuKeyValue("Auto-delete:", fmt.Sprintf("%v", autoDelete), false))
		if withRetry {
			maxRetries, _ := cmd.Flags().GetInt("max-retries")
			retryDelay, _ := cmd.Flags().GetInt("retry-delay")
			fmt.Print(ui.SubMenuKeyValue("Com Retry:", fmt.Sprintf("Sim (max: %d, delay: %ds)", maxRetries, retryDelay), false))
		}
		return nil
	}

	fmt.Println(ui.SubMenuLoading("Conectando ao RabbitMQ"))

	client, err := rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		fmt.Println(ui.SubMenuError("Erro ao conectar"))
		return fmt.Errorf("erro ao conectar: %w", err)
	}
	defer client.Close()

	exists, err := client.QueueExists(queueName)
	if err != nil {
		return fmt.Errorf("erro ao verificar fila: %w", err)
	}

	if exists {
		fmt.Println(ui.SubMenuWarning(fmt.Sprintf("Fila '%s' já existe", queueName)))

		var recreate bool
		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("🗑️  Recriar fila?").
					Description("A fila atual será deletada permanentemente").
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
			return fmt.Errorf("erro ao deletar fila: %w", err)
		}
		fmt.Println(ui.SubMenuDone("Fila deletada"))
	}

	fmt.Println(ui.SubMenuLoading("Criando fila"))

	opts := rabbitmq.CreateQueueOptions{
		Name:       queueName,
		Type:       queueType,
		Durable:    durable,
		AutoDelete: autoDelete,
		Exclusive:  false,
		NoWait:     false,
		Arguments:  make(map[string]interface{}),
	}

	if err := client.CreateQueue(opts); err != nil {
		fmt.Println(ui.SubMenuError("Erro ao criar fila"))
		return fmt.Errorf("erro ao criar fila: %w", err)
	}

	fmt.Println(ui.SubMenuDone("Fila criada com sucesso!"))
	fmt.Println()

	fmt.Println(ui.SubMenuSection("📋", "Detalhes da Fila"))
	fmt.Print(ui.SubMenuKeyValue("Nome:", queueName, true))
	fmt.Print(ui.SubMenuKeyValue("Tipo:", queueType, false))
	fmt.Print(ui.SubMenuKeyValue("Durável:", fmt.Sprintf("%v", durable), false))
	fmt.Print(ui.SubMenuKeyValue("Auto-delete:", fmt.Sprintf("%v", autoDelete), false))

	if withRetry {
		fmt.Println(ui.SubMenuWarning("Use 'gohop retry setup' para configurar retry"))
	}

	return nil
}

func runQueueList(cmd *cobra.Command, args []string) error {
	fmt.Print(ui.SubMenuHeader("📋", "Listar Filas", "Filas disponíveis no RabbitMQ"))

	cfg, err := config.Load(profile)
	if err != nil {
		fmt.Println(ui.SubMenuError("Erro ao carregar configuração"))
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	fmt.Println(ui.SubMenuLoading("Buscando filas"))

	mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
	queues, err := mgmtClient.ListQueues()
	if err != nil {
		fmt.Println(ui.SubMenuError("Erro ao listar filas"))
		return fmt.Errorf("erro ao listar filas: %w", err)
	}

	if len(queues) == 0 {
		fmt.Println(ui.SubMenuWarning("Nenhuma fila encontrada"))
		fmt.Println(ui.SubMenuHelp("Use 'gohop queue create <nome>' para criar uma fila"))
		return nil
	}

	fmt.Println(ui.SubMenuDone(fmt.Sprintf("%d fila(s) encontrada(s)", len(queues))))
	fmt.Println()

	// Se terminal interativo, usar tabela interativa
	if term.IsTerminal(int(os.Stdout.Fd())) && outputFmt == "table" {
		return ui.RunQueueTable(cfg)
	}

	// Modo não-interativo: tabela simples
	headers := []string{"Nome", "Tipo", "Msgs", "Unacked", "Consumers"}
	var rows [][]string
	for _, q := range queues {
		rows = append(rows, []string{
			truncateStr(q.Name, 35),
			q.Type,
			strconv.Itoa(q.MessagesReady),
			strconv.Itoa(q.MessagesUnacked),
			strconv.Itoa(q.Consumers),
		})
	}

	fmt.Print(ui.SubMenuTable(headers, rows))
	fmt.Println()

	return nil
}

func runQueueDelete(cmd *cobra.Command, args []string) error {
	queueName := args[0]
	fmt.Print(ui.SubMenuHeader("🗑️", "Deletar Fila", fmt.Sprintf("Removendo fila '%s'", queueName)))

	cfg, err := config.Load(profile)
	if err != nil {
		fmt.Println(ui.SubMenuError("Erro ao carregar configuração"))
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	cascade, _ := cmd.Flags().GetBool("cascade")

	mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
	queue, err := mgmtClient.GetQueue(cfg.RabbitMQ.VHost, queueName)
	if err != nil {
		fmt.Println(ui.SubMenuError("Fila não encontrada"))
		return fmt.Errorf("fila não encontrada: %s", queueName)
	}

	// Mostrar informações
	fmt.Println(ui.SubMenuSection("📊", "Informações da Fila"))
	fmt.Print(ui.SubMenuKeyValue("Nome:", queue.Name, true))
	fmt.Print(ui.SubMenuKeyValue("Mensagens prontas:", strconv.Itoa(queue.MessagesReady), queue.MessagesReady > 0))
	fmt.Print(ui.SubMenuKeyValue("Mensagens unacked:", strconv.Itoa(queue.MessagesUnacked), queue.MessagesUnacked > 0))
	fmt.Print(ui.SubMenuKeyValue("Consumers:", strconv.Itoa(queue.Consumers), queue.Consumers > 0))
	fmt.Println()

	// Aviso se tem dados
	if queue.MessagesReady > 0 || queue.MessagesUnacked > 0 || queue.Consumers > 0 {
		fmt.Println(ui.SubMenuWarning("A fila tem mensagens ou consumers ativos!"))
		fmt.Println()
	}

	var confirm bool
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("⚠️  Confirmar exclusão?").
				Description("Esta operação não pode ser desfeita").
				Value(&confirm),
		),
	)
	confirmForm.WithTheme(ui.GetCharmTheme())

	if err := confirmForm.Run(); err != nil {
		return err
	}

	if !confirm {
		fmt.Println(ui.SubMenuError("Operação cancelada"))
		return nil
	}

	// Deletar filas relacionadas (cascade)
	if cascade {
		fmt.Println(ui.SubMenuLoading("Deletando filas relacionadas"))
		waitQueue := fmt.Sprintf("%s.wait", queueName)
		dlqQueue := fmt.Sprintf("%s.dlq", queueName)

		for _, relatedQueue := range []string{waitQueue, dlqQueue} {
			_, err := mgmtClient.GetQueue(cfg.RabbitMQ.VHost, relatedQueue)
			if err == nil {
				if err := mgmtClient.DeleteQueueViaAPI(cfg.RabbitMQ.VHost, relatedQueue); err == nil {
					fmt.Println(ui.SubMenuDone(fmt.Sprintf("Fila '%s' deletada", relatedQueue)))
				}
			}
		}
	}

	fmt.Println(ui.SubMenuLoading("Deletando fila principal"))
	if err := mgmtClient.DeleteQueueViaAPI(cfg.RabbitMQ.VHost, queueName); err != nil {
		fmt.Println(ui.SubMenuError("Erro ao deletar fila"))
		return fmt.Errorf("erro ao deletar fila: %w", err)
	}

	fmt.Println(ui.SubMenuDone(fmt.Sprintf("Fila '%s' deletada com sucesso!", queueName)))

	return nil
}

func runQueueStatus(cmd *cobra.Command, args []string) error {
	queueName := args[0]
	fmt.Print(ui.SubMenuHeader("📊", "Status da Fila", fmt.Sprintf("Detalhes de '%s'", queueName)))

	cfg, err := config.Load(profile)
	if err != nil {
		fmt.Println(ui.SubMenuError("Erro ao carregar configuração"))
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
	queue, err := mgmtClient.GetQueue(cfg.RabbitMQ.VHost, queueName)
	if err != nil {
		fmt.Println(ui.SubMenuError("Fila não encontrada"))
		return fmt.Errorf("fila não encontrada: %s", queueName)
	}

	// Configuração
	fmt.Println(ui.SubMenuSection("⚙", "Configuração"))
	fmt.Print(ui.SubMenuKeyValue("Nome:", queue.Name, true))
	fmt.Print(ui.SubMenuKeyValue("Tipo:", queue.Type, false))
	fmt.Print(ui.SubMenuKeyValue("VHost:", queue.VHost, false))
	fmt.Print(ui.SubMenuKeyValue("Durável:", fmt.Sprintf("%v", queue.Durable), false))
	fmt.Print(ui.SubMenuKeyValue("Auto-delete:", fmt.Sprintf("%v", queue.AutoDelete), false))
	fmt.Print(ui.SubMenuKeyValue("Exclusive:", fmt.Sprintf("%v", queue.Exclusive), false))

	// Status de mensagens
	fmt.Println(ui.SubMenuSection("📨", "Mensagens"))

	// Indicador visual de status
	status := "success"
	if queue.MessagesReady > 100 {
		status = "warning"
	}
	if queue.MessagesReady > 1000 {
		status = "error"
	}

	fmt.Print(ui.SubMenuKeyValue("Prontas:", ui.SubMenuStatus(strconv.Itoa(queue.MessagesReady), status), false))
	fmt.Print(ui.SubMenuKeyValue("Unacked:", strconv.Itoa(queue.MessagesUnacked), queue.MessagesUnacked > 0))
	fmt.Print(ui.SubMenuKeyValue("Total:", strconv.Itoa(queue.Messages), false))

	// Consumers
	fmt.Println(ui.SubMenuSection("👥", "Consumers"))
	consumerStatus := "success"
	if queue.Consumers == 0 {
		consumerStatus = "warning"
	}
	fmt.Print(ui.SubMenuKeyValue("Ativos:", ui.SubMenuStatus(strconv.Itoa(queue.Consumers), consumerStatus), false))
	if queue.ConsumerUtilisation > 0 {
		fmt.Print(ui.SubMenuKeyValue("Utilização:", fmt.Sprintf("%.1f%%", queue.ConsumerUtilisation*100), false))
	}

	// Métricas
	if queue.MessageStats.Publish > 0 || queue.MessageStats.Deliver > 0 {
		fmt.Println(ui.SubMenuSection("📈", "Métricas"))
		fmt.Print(ui.SubMenuKeyValue("Publicações:", strconv.Itoa(queue.MessageStats.Publish), false))
		fmt.Print(ui.SubMenuKeyValue("Entregas:", strconv.Itoa(queue.MessageStats.Deliver), false))
		fmt.Print(ui.SubMenuKeyValue("Acknowledges:", strconv.Itoa(queue.MessageStats.Ack), false))
	}

	fmt.Println()
	return nil
}

func runQueuePurge(cmd *cobra.Command, args []string) error {
	queueName := args[0]
	fmt.Print(ui.SubMenuHeader("🧹", "Limpar Fila", fmt.Sprintf("Removendo mensagens de '%s'", queueName)))

	cfg, err := config.Load(profile)
	if err != nil {
		fmt.Println(ui.SubMenuError("Erro ao carregar configuração"))
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
	queue, err := mgmtClient.GetQueue(cfg.RabbitMQ.VHost, queueName)
	if err != nil {
		fmt.Println(ui.SubMenuError("Fila não encontrada"))
		return fmt.Errorf("fila não encontrada: %s", queueName)
	}

	if queue.MessagesReady == 0 {
		fmt.Println(ui.SubMenuInfo("A fila já está vazia"))
		return nil
	}

	fmt.Println(ui.SubMenuSection("📊", "Status Atual"))
	fmt.Print(ui.SubMenuKeyValue("Mensagens prontas:", strconv.Itoa(queue.MessagesReady), true))
	fmt.Println()

	fmt.Println(ui.SubMenuWarning(fmt.Sprintf("%d mensagens serão removidas permanentemente!", queue.MessagesReady)))
	fmt.Println()

	var confirm bool
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("⚠️  Confirmar limpeza?").
				Description("Esta operação não pode ser desfeita").
				Value(&confirm),
		),
	)
	confirmForm.WithTheme(ui.GetCharmTheme())

	if err := confirmForm.Run(); err != nil {
		return err
	}

	if !confirm {
		fmt.Println(ui.SubMenuError("Operação cancelada"))
		return nil
	}

	fmt.Println(ui.SubMenuLoading("Limpando fila"))

	client, err := rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		return fmt.Errorf("erro ao conectar: %w", err)
	}
	defer client.Close()

	messages, err := client.PurgeQueue(queueName, false)
	if err != nil {
		fmt.Println(ui.SubMenuError("Erro ao limpar fila"))
		return fmt.Errorf("erro ao limpar fila: %w", err)
	}

	fmt.Println(ui.SubMenuDone(fmt.Sprintf("%d mensagens removidas!", messages)))

	return nil
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
