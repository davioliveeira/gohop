package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/davioliveeira/rabbit/internal/config"
	"github.com/davioliveeira/rabbit/internal/rabbitmq"
	"github.com/spf13/cobra"
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
	// Flags para queue create
	queueCreateCmd.Flags().String("type", "quorum", "Tipo de fila (classic|quorum)")
	queueCreateCmd.Flags().Bool("durable", true, "Fila durável")
	queueCreateCmd.Flags().Bool("auto-delete", false, "Auto-deletar quando não usada")
	queueCreateCmd.Flags().Bool("with-retry", false, "Configurar sistema de retry automaticamente")
	queueCreateCmd.Flags().Int("max-retries", 3, "Número máximo de tentativas")
	queueCreateCmd.Flags().Int("retry-delay", 5, "Delay entre tentativas (segundos)")
	queueCreateCmd.Flags().Bool("dry-run", false, "Mostrar o que seria criado sem executar")

	// Flags para queue delete
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
	
	// Carregar configuração
	cfg, err := config.Load(profile)
	if err != nil {
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	// Obter flags
	queueType, _ := cmd.Flags().GetString("type")
	durable, _ := cmd.Flags().GetBool("durable")
	autoDelete, _ := cmd.Flags().GetBool("auto-delete")
	withRetry, _ := cmd.Flags().GetBool("with-retry")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if dryRun {
		fmt.Printf("\n[DRY RUN] Criaria fila:\n")
		fmt.Printf("  Nome:       %s\n", queueName)
		fmt.Printf("  Tipo:       %s\n", queueType)
		fmt.Printf("  Durable:    %v\n", durable)
		fmt.Printf("  AutoDelete: %v\n", autoDelete)
		if withRetry {
			maxRetries, _ := cmd.Flags().GetInt("max-retries")
			retryDelay, _ := cmd.Flags().GetInt("retry-delay")
			fmt.Printf("  Com Retry:  Sim (max: %d, delay: %ds)\n", maxRetries, retryDelay)
		}
		return nil
	}

	// Verificar se fila já existe
	client, err := rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		return fmt.Errorf("erro ao conectar: %w", err)
	}
	defer client.Close()

	exists, err := client.QueueExists(queueName)
	if err != nil {
		return fmt.Errorf("erro ao verificar se fila existe: %w", err)
	}

	if exists {
		// Perguntar confirmação se fila já existe
		var recreate bool
		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Fila '%s' já existe", queueName)).
					Description("Deseja recriar a fila? (a fila atual será deletada)").
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

		// Deletar fila existente
		_, err := client.DeleteQueue(queueName, false, false, false)
		if err != nil {
			return fmt.Errorf("erro ao deletar fila existente: %w", err)
		}
		fmt.Printf("🗑️  Fila '%s' deletada\n", queueName)
	}

	// Criar fila
	opts := rabbitmq.CreateQueueOptions{
		Name:       queueName,
		Type:       queueType,
		Durable:    durable,
		AutoDelete: autoDelete,
		Exclusive:  false,
		NoWait:     false,
		Arguments:  make(map[string]interface{}),
	}

	if withRetry {
		// Se with-retry, apenas criamos a fila básica agora
		// O setup completo de retry será na Fase 3
		fmt.Println("⚠️  --with-retry será implementado na Fase 3 (sistema de retry completo)")
	}

	if err := client.CreateQueue(opts); err != nil {
		return fmt.Errorf("erro ao criar fila: %w", err)
	}

	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("✅ Fila '%s' criada com sucesso!", queueName)))
	fmt.Printf("   Tipo:       %s\n", queueType)
	fmt.Printf("   Durable:    %v\n", durable)
	fmt.Printf("   AutoDelete: %v\n", autoDelete)

	return nil
}

func runQueueList(cmd *cobra.Command, args []string) error {
	// Carregar configuração
	cfg, err := config.Load(profile)
	if err != nil {
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	// Usar Management API para listar filas
	mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
	queues, err := mgmtClient.ListQueues()
	if err != nil {
		return fmt.Errorf("erro ao listar filas: %w", err)
	}

	if len(queues) == 0 {
		fmt.Println("ℹ️  Nenhuma fila encontrada")
		return nil
	}

	// Formatar como tabela
	if outputFmt == "json" {
		// TODO: Implementar output JSON
		fmt.Println("Output JSON ainda não implementado")
		return nil
	}

	// Tabela usando Lip Gloss
	fmt.Println()
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)
	fmt.Println(style.Render("📋 Filas Disponíveis"))
	fmt.Println()

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("213")).
		Bold(true).
		Width(40).
		Align(lipgloss.Left)

	rowStyle := lipgloss.NewStyle().
		Width(40).
		Align(lipgloss.Left)

	fmt.Printf("%s %s %s %s\n",
		headerStyle.Render("Nome"),
		headerStyle.Render("Tipo"),
		headerStyle.Render("Msgs"),
		headerStyle.Render("Consumers"))

	fmt.Println(strings.Repeat("-", 100))

	for _, queue := range queues {
		fmt.Printf("%s %s %s %s\n",
			rowStyle.Render(queue.Name),
			rowStyle.Render(queue.Type),
			rowStyle.Render(strconv.Itoa(queue.MessagesReady)),
			rowStyle.Render(strconv.Itoa(queue.Consumers)))
	}

	fmt.Println()
	fmt.Printf("Total: %d fila(s)\n", len(queues))

	return nil
}

func runQueueDelete(cmd *cobra.Command, args []string) error {
	queueName := args[0]
	
	// Carregar configuração
	cfg, err := config.Load(profile)
	if err != nil {
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	// Obter flags
	cascade, _ := cmd.Flags().GetBool("cascade")
	// Nota: ifUnused e ifEmpty serão usados na Fase 3 quando implementarmos delete via AMQP
	// Por enquanto, usamos Management API que não precisa dessas flags

	// Verificar se fila existe
	mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
	queue, err := mgmtClient.GetQueue(cfg.RabbitMQ.VHost, queueName)
	if err != nil {
		return fmt.Errorf("fila não encontrada: %s", queueName)
	}

	// Mostrar informações da fila
	fmt.Printf("\n📋 Informações da Fila:\n")
	fmt.Printf("   Nome:      %s\n", queue.Name)
	fmt.Printf("   Mensagens: %d (prontas) + %d (unacked)\n", queue.MessagesReady, queue.MessagesUnacked)
	fmt.Printf("   Consumers: %d\n", queue.Consumers)
	fmt.Println()

	// Se tem mensagens ou consumers, perguntar confirmação
	if queue.MessagesReady > 0 || queue.MessagesUnacked > 0 || queue.Consumers > 0 {
		var confirm bool
		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("⚠️  Atenção!").
					Description(fmt.Sprintf("A fila tem mensagens ou consumers ativos. Deletar mesmo assim?")).
					Value(&confirm),
			),
		)
		confirmForm.WithTheme(huh.ThemeCharm())
		
		if err := confirmForm.Run(); err != nil {
			return err
		}

		if !confirm {
			fmt.Println("❌ Operação cancelada")
			return nil
		}
	}

	// Deletar fila relacionada (cascade)
	if cascade {
		// Deletar wait queue, DLQ se existirem
		waitQueue := fmt.Sprintf("%s.wait", queueName)
		dlqQueue := fmt.Sprintf("%s.dlq", queueName)
		
		for _, relatedQueue := range []string{waitQueue, dlqQueue} {
			_, err := mgmtClient.GetQueue(cfg.RabbitMQ.VHost, relatedQueue)
			if err == nil {
				if err := mgmtClient.DeleteQueueViaAPI(cfg.RabbitMQ.VHost, relatedQueue); err == nil {
					fmt.Printf("🗑️  Fila relacionada '%s' deletada\n", relatedQueue)
				}
			}
		}
	}

	// Deletar via Management API (mais confiável)
	if err := mgmtClient.DeleteQueueViaAPI(cfg.RabbitMQ.VHost, queueName); err != nil {
		return fmt.Errorf("erro ao deletar fila: %w", err)
	}

	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)

	fmt.Println(successStyle.Render(fmt.Sprintf("✅ Fila '%s' deletada com sucesso!", queueName)))

	return nil
}

func runQueueStatus(cmd *cobra.Command, args []string) error {
	queueName := args[0]
	
	// Carregar configuração
	cfg, err := config.Load(profile)
	if err != nil {
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	// Obter informações da fila via Management API
	mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
	queue, err := mgmtClient.GetQueue(cfg.RabbitMQ.VHost, queueName)
	if err != nil {
		return fmt.Errorf("fila não encontrada: %s", queueName)
	}

	// Formatar saída
	fmt.Println()
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		PaddingBottom(1)

	fmt.Println(titleStyle.Render(fmt.Sprintf("📊 Status da Fila: %s", queue.Name)))
	fmt.Println()

	fmt.Printf("Configuração:\n")
	fmt.Printf("  Tipo:        %s\n", queue.Type)
	fmt.Printf("  VHost:       %s\n", queue.VHost)
	fmt.Printf("  Durable:     %v\n", queue.Durable)
	fmt.Printf("  AutoDelete:  %v\n", queue.AutoDelete)
	fmt.Printf("  Exclusive:   %v\n", queue.Exclusive)
	fmt.Println()

	fmt.Printf("Estatísticas:\n")
	fmt.Printf("  Mensagens Prontas:     %d\n", queue.MessagesReady)
	fmt.Printf("  Mensagens Unacked:     %d\n", queue.MessagesUnacked)
	fmt.Printf("  Total de Mensagens:    %d\n", queue.Messages)
	fmt.Printf("  Consumers:             %d\n", queue.Consumers)
	if queue.ConsumerUtilisation > 0 {
		fmt.Printf("  Utilização Consumer:  %.1f%%\n", queue.ConsumerUtilisation*100)
	}
	fmt.Println()

	if queue.MessageStats.Publish > 0 || queue.MessageStats.Deliver > 0 {
		fmt.Printf("Métricas (message_stats):\n")
		fmt.Printf("  Publicações:          %d\n", queue.MessageStats.Publish)
		fmt.Printf("  Entregas:             %d\n", queue.MessageStats.Deliver)
		fmt.Printf("  Acknowledges:         %d\n", queue.MessageStats.Ack)
		fmt.Println()
	}

	return nil
}

func runQueuePurge(cmd *cobra.Command, args []string) error {
	queueName := args[0]
	
	// Carregar configuração
	cfg, err := config.Load(profile)
	if err != nil {
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	// Verificar se fila existe e quantas mensagens tem
	mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
	queue, err := mgmtClient.GetQueue(cfg.RabbitMQ.VHost, queueName)
	if err != nil {
		return fmt.Errorf("fila não encontrada: %s", queueName)
	}

	if queue.MessagesReady == 0 {
		fmt.Println("ℹ️  A fila já está vazia")
		return nil
	}

	// Perguntar confirmação
	var confirm bool
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Limpar %d mensagens da fila '%s'?", queue.MessagesReady, queueName)).
				Description("Esta operação não pode ser desfeita").
				Value(&confirm),
		),
	)
	confirmForm.WithTheme(huh.ThemeCharm())
	
	if err := confirmForm.Run(); err != nil {
		return err
	}

	if !confirm {
		fmt.Println("❌ Operação cancelada")
		return nil
	}

	// Limpar fila
	client, err := rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		return fmt.Errorf("erro ao conectar: %w", err)
	}
	defer client.Close()

	messages, err := client.PurgeQueue(queueName, false)
	if err != nil {
		return fmt.Errorf("erro ao limpar fila: %w", err)
	}

	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)

	fmt.Println(successStyle.Render(fmt.Sprintf("✅ %d mensagens removidas da fila '%s'", messages, queueName)))

	return nil
}
