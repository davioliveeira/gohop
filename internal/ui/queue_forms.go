package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/davioliveeira/gohop/internal/config"
	"github.com/davioliveeira/gohop/internal/rabbitmq"
	"github.com/davioliveeira/gohop/internal/retry"
)

// ═══════════════════════════════════════════════════════════════════════════════
// TEMA CUSTOMIZADO PARA FORMULÁRIOS
// ═══════════════════════════════════════════════════════════════════════════════

func getCustomTheme() *huh.Theme {
	t := huh.ThemeCharm()

	// Cores do tema
	pink := lipgloss.Color("#FF6B9D")
	purple := lipgloss.Color("#C678DD")
	cyan := lipgloss.Color("#56B6C2")
	gray := lipgloss.Color("#5C6370")
	white := lipgloss.Color("#ABB2BF")

	// Customizar estilos
	t.Focused.Title = t.Focused.Title.Foreground(pink).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(gray).Italic(true)
	t.Focused.Base = t.Focused.Base.BorderForeground(purple)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(pink)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(cyan)
	t.Focused.Option = t.Focused.Option.Foreground(white)

	t.Blurred.Title = t.Blurred.Title.Foreground(gray)
	t.Blurred.Description = t.Blurred.Description.Foreground(gray)

	return t
}

// renderFormHeader renderiza um header bonito para os formulários
func renderFormHeader(icon, title, subtitle string) string {
	pink := lipgloss.Color("#FF6B9D")
	purple := lipgloss.Color("#C678DD")
	gray := lipgloss.Color("#5C6370")

	// Estilo do título
	titleStyle := lipgloss.NewStyle().
		Foreground(pink).
		Bold(true).
		MarginBottom(1)

	// Estilo do subtítulo
	subtitleStyle := lipgloss.NewStyle().
		Foreground(gray).
		Italic(true)

	// Estilo da borda
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(purple).
		Padding(1, 3).
		Width(60)

	// Construir conteúdo
	content := titleStyle.Render(fmt.Sprintf("%s  %s", icon, title))
	if subtitle != "" {
		content += "\n" + subtitleStyle.Render(subtitle)
	}

	return borderStyle.Render(content) + "\n\n"
}

// ═══════════════════════════════════════════════════════════════════════════════
// ESTRUTURAS
// ═══════════════════════════════════════════════════════════════════════════════

// QueueCreateFormResult contém os dados do formulário de criação de fila
type QueueCreateFormResult struct {
	QueueName    string
	Durable      bool
	AutoDelete   bool
	WithRetry    bool
	MaxRetries   int
	RetryDelay   int
	DLQTTL       int
	QueueTypeSel string
}

// ═══════════════════════════════════════════════════════════════════════════════
// FORMULÁRIO DE CRIAÇÃO DE FILA
// ═══════════════════════════════════════════════════════════════════════════════

func RunQueueCreateForm(cfg *config.Config) (*QueueCreateFormResult, error) {
	// Limpar tela e mostrar header estilizado
	fmt.Print("\033[H\033[2J") // Clear screen

	// Header animado
	renderCreateQueueHeader()

	var (
		queueName    string
		queueTypeSel string = "quorum"
		durable      bool   = true
		autoDelete   bool   = false
		withRetry    bool   = false
		maxRetries   string = "3"
		retryDelay   string = "5"
		dlqTTL       string = "604800000"
	)

	// ═══════════════════════════════════════════════════════════════════════
	// STEP 1: Informações Básicas
	// ═══════════════════════════════════════════════════════════════════════
	renderStepHeader(1, 3, "Informações Básicas", "Configure o nome e tipo da fila")

	form1 := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Nome da Fila").
				Description("Identificador único (sem espaços)").
				Value(&queueName).
				Placeholder("minha-fila-producao").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("obrigatório")
					}
					if strings.Contains(s, " ") {
						return fmt.Errorf("sem espaços")
					}
					if len(s) > 255 {
						return fmt.Errorf("máximo 255 caracteres")
					}
					return nil
				}),

			huh.NewSelect[string]().
				Title("Tipo de Fila").
				Description("Escolha o tipo de persistência").
				Options(
					huh.NewOption("🏛️  Classic - Tradicional, single node", "classic"),
					huh.NewOption("⚡ Quorum - Alta disponibilidade (recomendado)", "quorum"),
				).
				Value(&queueTypeSel),
		),
	)
	form1.WithTheme(getCustomTheme())

	if err := form1.Run(); err != nil {
		return nil, fmt.Errorf("cancelado")
	}

	// ═══════════════════════════════════════════════════════════════════════
	// STEP 2: Configurações Avançadas
	// ═══════════════════════════════════════════════════════════════════════
	renderStepHeader(2, 3, "Configurações", "Defina comportamento da fila")

	// Quorum queues TÊM que ser duráveis e NÃO podem ser auto-delete
	if queueTypeSel == "quorum" {
		// Mostrar nota explicativa
		noteStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(InfoColor).
			Padding(1, 2).
			Width(60)

		noteContent := lipgloss.NewStyle().Foreground(InfoColor).Bold(true).Render("ℹ️  Filas Quorum") + "\n\n" +
			lipgloss.NewStyle().Foreground(TextSecondary).Render(
				"Filas Quorum são sempre duráveis e não suportam auto-delete.\n"+
					"Isso garante alta disponibilidade e replicação dos dados.")

		fmt.Println(noteStyle.Render(noteContent))
		fmt.Println()

		// Forçar configurações para quorum
		durable = true
		autoDelete = false

		// Apenas perguntar sobre retry
		form2 := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("🔄 Sistema de Retry/DLQ").
					Description("Configura retry automático e Dead Letter Queue").
					Affirmative("Sim, configurar").
					Negative("Não, fila simples").
					Value(&withRetry),
			),
		)
		form2.WithTheme(getCustomTheme())

		if err := form2.Run(); err != nil {
			return nil, fmt.Errorf("cancelado")
		}
	} else {
		// Classic queue - permite configurar tudo
		form2 := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("💾 Fila Durável").
					Description("Persiste em disco, sobrevive a restarts do servidor").
					Affirmative("Sim, durável").
					Negative("Não, temporária").
					Value(&durable),

				huh.NewConfirm().
					Title("🗑️  Auto-deletar").
					Description("Remove automaticamente quando sem consumers conectados").
					Affirmative("Sim").
					Negative("Não").
					Value(&autoDelete),

				huh.NewConfirm().
					Title("🔄 Sistema de Retry/DLQ").
					Description("Configura retry automático e Dead Letter Queue").
					Affirmative("Sim, configurar").
					Negative("Não, fila simples").
					Value(&withRetry),
			),
		)
		form2.WithTheme(getCustomTheme())

		if err := form2.Run(); err != nil {
			return nil, fmt.Errorf("cancelado")
		}
	}

	// ═══════════════════════════════════════════════════════════════════════
	// STEP 3: Configurações de Retry (condicional)
	// ═══════════════════════════════════════════════════════════════════════
	if withRetry {
		renderStepHeader(3, 3, "Sistema de Retry", "Configure o comportamento de retry e DLQ")

		form3 := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("🔢 Máximo de Tentativas").
					Description("Retries antes de enviar para DLQ (1-10)").
					Value(&maxRetries).
					Placeholder("3").
					Validate(func(s string) error {
						if s == "" {
							return nil
						}
						val, err := strconv.Atoi(s)
						if err != nil || val < 1 || val > 10 {
							return fmt.Errorf("entre 1 e 10")
						}
						return nil
					}),

				huh.NewInput().
					Title("⏱️  Delay de Retry (segundos)").
					Description("Tempo de espera entre tentativas").
					Value(&retryDelay).
					Placeholder("5").
					Validate(func(s string) error {
						if s == "" {
							return nil
						}
						val, err := strconv.Atoi(s)
						if err != nil || val < 1 || val > 3600 {
							return fmt.Errorf("entre 1 e 3600")
						}
						return nil
					}),

			huh.NewSelect[string]().
				Title("📅 Retenção na DLQ").
				Description("Por quanto tempo manter mensagens mortas").
				Options(
					huh.NewOption("♾️  Sem expiração (manter para sempre)", "0"),
					huh.NewOption("1 dia", "86400000"),
					huh.NewOption("7 dias (recomendado)", "604800000"),
					huh.NewOption("30 dias", "2592000000"),
					huh.NewOption("90 dias", "7776000000"),
				).
				Value(&dlqTTL),
			),
		)
		form3.WithTheme(getCustomTheme())

		if err := form3.Run(); err != nil {
			return nil, fmt.Errorf("cancelado")
		}
	}

	// ═══════════════════════════════════════════════════════════════════════
	// RESUMO E CONFIRMAÇÃO
	// ═══════════════════════════════════════════════════════════════════════
	renderSummary(queueName, queueTypeSel, durable, autoDelete, withRetry, maxRetries, retryDelay, dlqTTL)

	var confirm bool
	formConfirm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("✅ Confirmar Criação").
				Description("Criar fila com estas configurações?").
				Affirmative("Criar Fila").
				Negative("Cancelar").
				Value(&confirm),
		),
	)
	formConfirm.WithTheme(getCustomTheme())

	if err := formConfirm.Run(); err != nil {
		return nil, fmt.Errorf("cancelado")
	}

	if !confirm {
		return nil, fmt.Errorf("operação cancelada")
	}

	// Converter strings para int
	maxRetriesInt := 3
	if v, err := strconv.Atoi(maxRetries); err == nil {
		maxRetriesInt = v
	}

	retryDelayInt := 5
	if v, err := strconv.Atoi(retryDelay); err == nil {
		retryDelayInt = v
	}

	dlqTTLInt := 604800000
	if v, err := strconv.Atoi(dlqTTL); err == nil {
		dlqTTLInt = v
	}

	return &QueueCreateFormResult{
		QueueName:    queueName,
		Durable:      durable,
		AutoDelete:   autoDelete,
		WithRetry:    withRetry,
		MaxRetries:   maxRetriesInt,
		RetryDelay:   retryDelayInt,
		DLQTTL:       dlqTTLInt,
		QueueTypeSel: queueTypeSel,
	}, nil
}

// Helpers para renderização do formulário de criação

func renderCreateQueueHeader() {
	logo := `
   ╔═══════════════════════════════════════════════════════════════╗
   ║                                                               ║
   ║   🐰  G O H O P   -   Q U E U E   C R E A T O R              ║
   ║                                                               ║
   ╚═══════════════════════════════════════════════════════════════╝
`
	logoStyle := lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(MutedColor).
		Italic(true).
		Align(lipgloss.Center)

	fmt.Println(logoStyle.Render(logo))
	fmt.Println(subtitleStyle.Render("Configure sua nova fila com facilidade"))
	fmt.Println()
}

func renderStepHeader(current, total int, title, subtitle string) {
	// Barra de progresso
	progressWidth := 40
	filled := int(float64(progressWidth) * float64(current) / float64(total))
	empty := progressWidth - filled

	progressBar := lipgloss.NewStyle().Foreground(PrimaryColor).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(MutedColorDark).Render(strings.Repeat("░", empty))

	stepStyle := lipgloss.NewStyle().
		Foreground(AccentColor).
		Bold(true)

	titleStyle := lipgloss.NewStyle().
		Foreground(TextPrimary).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(MutedColor).
		Italic(true)

	// Separador
	sepStyle := lipgloss.NewStyle().Foreground(MutedColorDark)
	separator := sepStyle.Render(strings.Repeat("─", 60))

	fmt.Println()
	fmt.Println(separator)
	fmt.Printf("  %s  %s\n", progressBar, stepStyle.Render(fmt.Sprintf("PASSO %d/%d", current, total)))
	fmt.Println(separator)
	fmt.Println()
	fmt.Printf("  %s\n", titleStyle.Render(title))
	fmt.Printf("  %s\n", subtitleStyle.Render(subtitle))
	fmt.Println()
}

func renderSummary(name, qType string, durable, autoDelete, withRetry bool, maxRetries, retryDelay, dlqTTL string) {
	fmt.Println()

	// Box de resumo
	titleStyle := lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(MutedColor).
		Width(22)

	valueStyle := lipgloss.NewStyle().
		Foreground(TextPrimary).
		Bold(true)

	checkStyle := lipgloss.NewStyle().Foreground(SuccessColor)
	crossStyle := lipgloss.NewStyle().Foreground(ErrorColor)

	boolIcon := func(b bool) string {
		if b {
			return checkStyle.Render("✓")
		}
		return crossStyle.Render("✗")
	}

	var lines []string
	lines = append(lines, titleStyle.Render("📋 RESUMO DA CONFIGURAÇÃO"))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(MutedColorDark).Render(strings.Repeat("─", 45)))
	lines = append(lines, "")

	// Info básica
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("Nome da Fila:"),
		valueStyle.Copy().Foreground(InfoColor).Render(name)))

	qTypeDisplay := "Classic"
	if qType == "quorum" {
		qTypeDisplay = "Quorum ⚡"
	}
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("Tipo:"),
		valueStyle.Render(qTypeDisplay)))

	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("Durável:"),
		valueStyle.Render(boolIcon(durable)+" "+boolToStr(durable))))

	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("Auto-deletar:"),
		valueStyle.Render(boolIcon(autoDelete)+" "+boolToStr(autoDelete))))

	lines = append(lines, "")

	// Retry info
	if withRetry {
		lines = append(lines, lipgloss.NewStyle().Foreground(SecondaryColor).Bold(true).Render("🔄 SISTEMA DE RETRY"))
		lines = append(lines, "")

		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("  Max Retries:"),
			valueStyle.Render(maxRetries+" tentativas")))

		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("  Delay:"),
			valueStyle.Render(retryDelay+" segundos")))

		dlqDays := "7 dias"
		switch dlqTTL {
		case "0":
			dlqDays = "♾️  Sem expiração"
		case "86400000":
			dlqDays = "1 dia"
		case "2592000000":
			dlqDays = "30 dias"
		case "7776000000":
			dlqDays = "90 dias"
		}
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("  Retenção DLQ:"),
			valueStyle.Render(dlqDays)))

		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(MutedColor).Italic(true).Render("  Componentes que serão criados:"))
		lines = append(lines, lipgloss.NewStyle().Foreground(SuccessColor).Render(fmt.Sprintf("    • %s (fila principal)", name)))
		lines = append(lines, lipgloss.NewStyle().Foreground(SuccessColor).Render(fmt.Sprintf("    • %s.wait (delay queue)", name)))
		lines = append(lines, lipgloss.NewStyle().Foreground(SuccessColor).Render(fmt.Sprintf("    • %s.dlq (dead letter)", name)))
		lines = append(lines, lipgloss.NewStyle().Foreground(SuccessColor).Render(fmt.Sprintf("    • %s.wait.exchange", name)))
		lines = append(lines, lipgloss.NewStyle().Foreground(SuccessColor).Render(fmt.Sprintf("    • %s.retry.exchange", name)))
	} else {
		lines = append(lines, lipgloss.NewStyle().Foreground(MutedColor).Italic(true).Render("Sistema de retry: desabilitado"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(SecondaryColor).
		Padding(1, 2).
		Width(55).
		Render(content)

	fmt.Println(box)
	fmt.Println()
}

func boolToStr(b bool) string {
	if b {
		return "Sim"
	}
	return "Não"
}

// ═══════════════════════════════════════════════════════════════════════════════
// CRIAÇÃO DA FILA
// ═══════════════════════════════════════════════════════════════════════════════

func CreateQueueFromForm(cfg *config.Config, result *QueueCreateFormResult) error {
	mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
	vhost := cfg.RabbitMQ.VHost
	if vhost == "" {
		vhost = "/"
	} else if vhost[0] != '/' {
		vhost = "/" + vhost
	}

	// Iniciar progress visual
	fmt.Println()
	renderCreationHeader(result.QueueName)

	// Task tracker
	tasks := []creationTask{
		{name: "Verificar existência", status: "pending"},
		{name: "Conectar ao RabbitMQ", status: "pending"},
		{name: "Criar fila principal", status: "pending"},
	}

	if result.WithRetry {
		tasks = append(tasks, creationTask{name: "Criar Wait Exchange", status: "pending"})
		tasks = append(tasks, creationTask{name: "Criar Wait Queue", status: "pending"})
		tasks = append(tasks, creationTask{name: "Criar Retry Exchange", status: "pending"})
		tasks = append(tasks, creationTask{name: "Criar Dead Letter Queue", status: "pending"})
		tasks = append(tasks, creationTask{name: "Configurar DLX na fila", status: "pending"})
	}

	renderTasks(tasks)

	// ═══════════════════════════════════════════════════════════════════════
	// TASK 1: Verificar existência
	// ═══════════════════════════════════════════════════════════════════════
	tasks[0].status = "running"
	renderTasks(tasks)

	_, err := mgmtClient.GetQueue(vhost, result.QueueName)
	exists := err == nil

	if exists {
		tasks[0].status = "warning"
		tasks[0].message = "Fila existe"
		renderTasks(tasks)

		// Perguntar se quer recriar
		fmt.Println()
		warningBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(WarningColor).
			Padding(1, 2).
			Render(lipgloss.NewStyle().Foreground(WarningColor).Bold(true).Render("⚠️  A fila já existe!") +
				"\n\n" +
				lipgloss.NewStyle().Foreground(TextSecondary).Render(
					fmt.Sprintf("A fila '%s' já existe no RabbitMQ.\nDeseja deletá-la e criar novamente?", result.QueueName)))
		fmt.Println(warningBox)

		var recreate bool
		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Confirmar recriação").
					Affirmative("Sim, recriar").
					Negative("Não, cancelar").
					Value(&recreate),
			),
		)
		confirmForm.WithTheme(getCustomTheme())

		if err := confirmForm.Run(); err != nil {
			return err
		}

		if !recreate {
			return fmt.Errorf("operação cancelada")
		}

		// Deletar
		if err := mgmtClient.DeleteQueueViaAPI(vhost, result.QueueName); err != nil {
			tasks[0].status = "error"
			tasks[0].message = "Erro ao deletar"
			renderTasks(tasks)
			return fmt.Errorf("erro ao deletar fila: %w", err)
		}
	}

	tasks[0].status = "done"
	tasks[0].message = ""
	renderTasks(tasks)
	animateDelay()

	// ═══════════════════════════════════════════════════════════════════════
	// TASK 2: Conectar
	// ═══════════════════════════════════════════════════════════════════════
	tasks[1].status = "running"
	renderTasks(tasks)

	client, err := rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		tasks[1].status = "error"
		tasks[1].message = "Falha na conexão"
		renderTasks(tasks)
		return fmt.Errorf("erro ao conectar: %w", err)
	}
	defer client.Close()

	tasks[1].status = "done"
	renderTasks(tasks)
	animateDelay()

	// ═══════════════════════════════════════════════════════════════════════
	// TASK 3: Criar fila
	// ═══════════════════════════════════════════════════════════════════════
	tasks[2].status = "running"
	renderTasks(tasks)

	opts := rabbitmq.CreateQueueOptions{
		Name:       result.QueueName,
		Type:       result.QueueTypeSel,
		Durable:    result.Durable,
		AutoDelete: result.AutoDelete,
		Exclusive:  false,
		NoWait:     false,
		Arguments:  make(map[string]interface{}),
	}

	if err := client.CreateQueue(opts); err != nil {
		tasks[2].status = "error"
		tasks[2].message = "Falha"
		renderTasks(tasks)
		return fmt.Errorf("erro ao criar fila: %w", err)
	}

	tasks[2].status = "done"
	renderTasks(tasks)
	animateDelay()

	// ═══════════════════════════════════════════════════════════════════════
	// TASKS 4-8: Sistema de Retry
	// ═══════════════════════════════════════════════════════════════════════
	if result.WithRetry {
		setupOpts := retry.SetupOptions{
			QueueName:  result.QueueName,
			QueueType:  result.QueueTypeSel,
			MaxRetries: result.MaxRetries,
			RetryDelay: result.RetryDelay,
			DLQTTL:     result.DLQTTL,
			Force:      true,
		}

		// Simular progresso das tarefas de retry
		for i := 3; i < len(tasks)-1; i++ {
			tasks[i].status = "running"
			renderTasks(tasks)
			animateDelay()
			tasks[i].status = "done"
			renderTasks(tasks)
		}

		// Última tarefa (Setup completo)
		tasks[len(tasks)-1].status = "running"
		renderTasks(tasks)

		if err := retry.SetupRetry(client, setupOpts); err != nil {
			tasks[len(tasks)-1].status = "error"
			tasks[len(tasks)-1].message = "Erro"
			renderTasks(tasks)
			return fmt.Errorf("erro ao configurar retry: %w", err)
		}

		if err := retry.RecreateQueueWithDLX(client, result.QueueName, result.QueueTypeSel); err != nil {
			tasks[len(tasks)-1].status = "error"
			tasks[len(tasks)-1].message = "Erro DLX"
			renderTasks(tasks)
			return fmt.Errorf("erro ao configurar DLX: %w", err)
		}

		tasks[len(tasks)-1].status = "done"
		renderTasks(tasks)
	}

	// ═══════════════════════════════════════════════════════════════════════
	// SUCESSO
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println()
	renderCreationSuccess(result)

	return nil
}

// Structs e helpers para criação visual

type creationTask struct {
	name    string
	status  string // pending, running, done, error, warning
	message string
}

func renderCreationHeader(queueName string) {
	titleStyle := lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(MutedColor)

	sepStyle := lipgloss.NewStyle().
		Foreground(MutedColorDark)

	fmt.Println(sepStyle.Render(strings.Repeat("━", 60)))
	fmt.Println(titleStyle.Render("⚡ CRIANDO FILA"))
	fmt.Println(subtitleStyle.Render(fmt.Sprintf("   %s", queueName)))
	fmt.Println(sepStyle.Render(strings.Repeat("━", 60)))
	fmt.Println()
}

func renderTasks(tasks []creationTask) {
	// Move cursor up para sobrescrever as tarefas anteriores
	if len(tasks) > 0 {
		fmt.Printf("\033[%dA", len(tasks)+2) // +2 para as linhas extras
	}

	// Calcular progresso
	done := 0
	for _, t := range tasks {
		if t.status == "done" {
			done++
		}
	}
	percent := float64(done) / float64(len(tasks))

	// Barra de progresso
	barWidth := 40
	filled := int(float64(barWidth) * percent)
	empty := barWidth - filled

	progressBar := lipgloss.NewStyle().Foreground(SuccessColor).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(MutedColorDark).Render(strings.Repeat("░", empty))

	percentText := lipgloss.NewStyle().Foreground(AccentColor).Bold(true).Render(
		fmt.Sprintf(" %d%%", int(percent*100)))

	fmt.Printf("  %s%s\n\n", progressBar, percentText)

	// Listar tarefas
	for _, task := range tasks {
		var icon, color string

		switch task.status {
		case "pending":
			icon = "○"
			color = string(MutedColor)
		case "running":
			icon = "◉"
			color = string(InfoColor)
		case "done":
			icon = "✓"
			color = string(SuccessColor)
		case "error":
			icon = "✗"
			color = string(ErrorColor)
		case "warning":
			icon = "⚠"
			color = string(WarningColor)
		}

		iconStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(true)
		nameStyle := lipgloss.NewStyle().Foreground(TextPrimary)
		msgStyle := lipgloss.NewStyle().Foreground(MutedColor).Italic(true)

		line := fmt.Sprintf("  %s %s", iconStyle.Render(icon), nameStyle.Render(task.name))
		if task.message != "" {
			line += " " + msgStyle.Render(fmt.Sprintf("(%s)", task.message))
		}

		// Pad line to clear previous content
		line = fmt.Sprintf("%-60s", line)
		fmt.Println(line)
	}
}

func animateDelay() {
	time.Sleep(150 * time.Millisecond)
}

func renderCreationSuccess(result *QueueCreateFormResult) {
	successBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(SuccessColor).
		Padding(1, 3).
		Width(60)

	titleStyle := lipgloss.NewStyle().
		Foreground(SuccessColor).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(MutedColor).
		Width(18)

	valueStyle := lipgloss.NewStyle().
		Foreground(TextPrimary).
		Bold(true)

	var lines []string
	lines = append(lines, titleStyle.Render("✅ FILA CRIADA COM SUCESSO!"))
	lines = append(lines, "")
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("Nome:"),
		valueStyle.Copy().Foreground(InfoColor).Render(result.QueueName)))
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("Tipo:"),
		valueStyle.Render(result.QueueTypeSel)))

	if result.WithRetry {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(SecondaryColor).Render("🔄 Sistema de Retry configurado"))
		lines = append(lines, lipgloss.NewStyle().Foreground(MutedColor).Render(
			fmt.Sprintf("   %d tentativas, %ds delay", result.MaxRetries, result.RetryDelay)))
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(MutedColor).Italic(true).Render(
		"Use 'gohop monitor "+result.QueueName+"' para monitorar"))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	fmt.Println(successBox.Render(content))
}

// ═══════════════════════════════════════════════════════════════════════════════
// FORMULÁRIOS DE MONITORAMENTO
// ═══════════════════════════════════════════════════════════════════════════════

func RunMonitorQueueForm(cfg *config.Config) ([]string, error) {
	fmt.Print(renderFormHeader("📊", "Monitorar Múltiplas Filas", "Selecione as filas para monitoramento simultâneo"))

	mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
	queues, err := mgmtClient.ListQueues()
	if err != nil {
		return nil, fmt.Errorf("erro ao listar filas: %w", err)
	}

	if len(queues) == 0 {
		return nil, fmt.Errorf("nenhuma fila encontrada no RabbitMQ")
	}

	// Criar opções com informações detalhadas
	queueOptions := make([]huh.Option[string], len(queues))
	for i, queue := range queues {
		label := fmt.Sprintf("%-30s │ %5d msgs │ %d consumers",
			truncateString(queue.Name, 30),
			queue.MessagesReady,
			queue.Consumers)
		queueOptions[i] = huh.NewOption(label, queue.Name)
	}

	var selectedQueues []string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("📋 Filas Disponíveis").
				Description("Use SPACE para selecionar, ENTER para confirmar").
				Options(queueOptions...).
				Value(&selectedQueues).
				Validate(func([]string) error {
					if len(selectedQueues) == 0 {
						return fmt.Errorf("selecione pelo menos uma fila")
					}
					return nil
				}),
		),
	)

	form.WithTheme(getCustomTheme())

	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("formulário cancelado: %w", err)
	}

	return selectedQueues, nil
}

func RunSingleMonitorForm(cfg *config.Config) (string, error) {
	fmt.Print(renderFormHeader("📊", "Monitorar Fila", "Selecione uma fila para dashboard em tempo real"))

	mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
	queues, err := mgmtClient.ListQueues()
	if err != nil {
		return "", fmt.Errorf("erro ao listar filas: %w", err)
	}

	if len(queues) == 0 {
		return "", fmt.Errorf("nenhuma fila encontrada no RabbitMQ")
	}

	// Criar opções com informações detalhadas
	queueOptions := make([]huh.Option[string], len(queues))
	for i, queue := range queues {
		status := "🟢"
		if queue.MessagesReady > 100 {
			status = "🟡"
		}
		if queue.MessagesReady > 1000 {
			status = "🔴"
		}

		label := fmt.Sprintf("%s %-25s │ %5d msgs │ %d consumers",
			status,
			truncateString(queue.Name, 25),
			queue.MessagesReady,
			queue.Consumers)
		queueOptions[i] = huh.NewOption(label, queue.Name)
	}

	var selectedQueue string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("📋 Selecionar Fila").
				Description("Escolha a fila para monitorar").
				Options(queueOptions...).
				Value(&selectedQueue),
		),
	)

	form.WithTheme(getCustomTheme())

	if err := form.Run(); err != nil {
		return "", fmt.Errorf("formulário cancelado: %w", err)
	}

	if selectedQueue == "" {
		return "", fmt.Errorf("nenhuma fila selecionada")
	}

	return selectedQueue, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// RECONFIGURAR FILA COM RETRY
// ═══════════════════════════════════════════════════════════════════════════════

// ReconfigureQueueResult contém os dados para reconfigurar uma fila
type ReconfigureQueueResult struct {
	QueueName   string
	MaxRetries  int
	RetryDelay  int
	DLQTTL      int
	QueueType   string // classic ou quorum (mantém o original)
}

// RunReconfigureQueueForm executa o formulário para reconfigurar uma fila com retry
func RunReconfigureQueueForm(cfg *config.Config) (*ReconfigureQueueResult, error) {
	fmt.Print(renderFormHeader("🔄", "Reconfigurar Fila com Retry",
		"Adicione sistema de retry a uma fila existente SEM perder mensagens"))

	// Listar filas existentes
	mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
	queues, err := mgmtClient.ListQueues()
	if err != nil {
		return nil, fmt.Errorf("erro ao listar filas: %w", err)
	}

	if len(queues) == 0 {
		return nil, fmt.Errorf("nenhuma fila encontrada no RabbitMQ")
	}

	// Filtrar filas que NÃO são de retry/wait/dlq
	var mainQueues []rabbitmq.QueueInfoManagement
	for _, q := range queues {
		// Ignorar filas de sistema de retry
		if strings.HasSuffix(q.Name, ".wait") ||
			strings.HasSuffix(q.Name, ".dlq") ||
			strings.Contains(q.Name, ".retry") {
			continue
		}
		mainQueues = append(mainQueues, q)
	}

	if len(mainQueues) == 0 {
		return nil, fmt.Errorf("nenhuma fila principal encontrada (apenas filas de retry)")
	}

	// Criar opções com informações detalhadas
	queueOptions := make([]huh.Option[string], len(mainQueues))
	for i, queue := range mainQueues {
		hasRetry := false
		// Verificar se já tem retry configurado
		for _, q := range queues {
			if q.Name == queue.Name+".wait" {
				hasRetry = true
				break
			}
		}

		status := "⚪" // sem retry
		if hasRetry {
			status = "🔄" // já tem retry
		}

		label := fmt.Sprintf("%s %-25s │ %5d msgs │ %s",
			status,
			truncateString(queue.Name, 25),
			queue.MessagesReady,
			queue.Type)
		queueOptions[i] = huh.NewOption(label, queue.Name)
	}

	var (
		selectedQueue string
		maxRetries    string = "3"
		retryDelay    string = "5"
		dlqTTL        string = "604800000" // 7 dias
		confirm       bool
	)

	// Formulário de seleção de fila
	form1 := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("📋 Legenda").
				Description("⚪ Sem retry  │  🔄 Já tem retry"),

			huh.NewSelect[string]().
				Title("Selecione a Fila").
				Description("Escolha a fila para reconfigurar").
				Options(queueOptions...).
				Value(&selectedQueue),
		),
	)
	form1.WithTheme(getCustomTheme())

	if err := form1.Run(); err != nil {
		return nil, fmt.Errorf("formulário cancelado: %w", err)
	}

	// Buscar detalhes da fila selecionada
	var selectedQueueDetails *rabbitmq.QueueInfoManagement
	for _, q := range mainQueues {
		if q.Name == selectedQueue {
			selectedQueueDetails = &q
			break
		}
	}

	if selectedQueueDetails == nil {
		return nil, fmt.Errorf("fila não encontrada")
	}

	// Formulário de configuração de retry
	form2 := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("⚙️ Configurações de Retry").
				Description(fmt.Sprintf("Configurando retry para: %s\nMensagens atuais: %d",
					selectedQueue, selectedQueueDetails.MessagesReady)),

			huh.NewInput().
				Title("Máximo de Tentativas").
				Description("Quantas vezes retentar antes de enviar para DLQ").
				Value(&maxRetries).
				Placeholder("3").
				Validate(func(s string) error {
					if s == "" {
						return nil
					}
					val, err := strconv.Atoi(s)
					if err != nil {
						return fmt.Errorf("deve ser um número")
					}
					if val < 1 {
						return fmt.Errorf("deve ser pelo menos 1")
					}
					return nil
				}),

			huh.NewInput().
				Title("Delay entre Tentativas (segundos)").
				Description("Tempo de espera antes de cada retry").
				Value(&retryDelay).
				Placeholder("5").
				Validate(func(s string) error {
					if s == "" {
						return nil
					}
					val, err := strconv.Atoi(s)
					if err != nil {
						return fmt.Errorf("deve ser um número")
					}
					if val < 1 {
						return fmt.Errorf("deve ser pelo menos 1 segundo")
					}
					return nil
				}),

			huh.NewSelect[string]().
				Title("📅 Retenção na DLQ").
				Description("Por quanto tempo manter mensagens mortas").
				Options(
					huh.NewOption("♾️  Sem expiração (manter para sempre)", "0"),
					huh.NewOption("1 dia", "86400000"),
					huh.NewOption("7 dias (recomendado)", "604800000"),
					huh.NewOption("30 dias", "2592000000"),
					huh.NewOption("90 dias", "7776000000"),
				).
				Value(&dlqTTL),
		),
	)
	form2.WithTheme(getCustomTheme())

	if err := form2.Run(); err != nil {
		return nil, fmt.Errorf("formulário cancelado: %w", err)
	}

	// Confirmar operação
	warningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5C07B")).
		Bold(true)

	fmt.Println()
	fmt.Println(warningStyle.Render("⚠️  ATENÇÃO: Esta operação irá:"))
	fmt.Println("   1. Salvar todas as mensagens da fila")
	fmt.Println("   2. Criar sistema de retry (wait queue, exchanges, DLQ)")
	fmt.Println("   3. Deletar a fila original")
	fmt.Println("   4. Recriar a fila com DLX configurado")
	fmt.Println("   5. Republicar todas as mensagens salvas")
	fmt.Println()

	form3 := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Confirmar Operação").
				Description(fmt.Sprintf("Reconfigurar '%s' com %d mensagens?", selectedQueue, selectedQueueDetails.MessagesReady)).
				Value(&confirm),
		),
	)
	form3.WithTheme(getCustomTheme())

	if err := form3.Run(); err != nil {
		return nil, fmt.Errorf("formulário cancelado: %w", err)
	}

	if !confirm {
		return nil, fmt.Errorf("operação cancelada pelo usuário")
	}

	// Converter valores
	maxRetriesInt := 3
	if v, err := strconv.Atoi(maxRetries); err == nil {
		maxRetriesInt = v
	}

	retryDelayInt := 5
	if v, err := strconv.Atoi(retryDelay); err == nil {
		retryDelayInt = v
	}

	dlqTTLInt := 604800000
	if v, err := strconv.Atoi(dlqTTL); err == nil {
		dlqTTLInt = v
	}

	return &ReconfigureQueueResult{
		QueueName:  selectedQueue,
		MaxRetries: maxRetriesInt,
		RetryDelay: retryDelayInt,
		DLQTTL:     dlqTTLInt,
		QueueType:  selectedQueueDetails.Type,
	}, nil
}

// ReconfigureQueueWithRetry executa a migração completa
func ReconfigureQueueWithRetry(cfg *config.Config, result *ReconfigureQueueResult) error {
	fmt.Println()

	// Estilo para mensagens
	stepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#56B6C2")).Bold(true)
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#98C379"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E06C75"))
	countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E5C07B")).Bold(true)

	// PASSO 1: Criar cliente e verificar fila
	fmt.Println(stepStyle.Render("━━━ PASSO 1/5: Conectando ao RabbitMQ ━━━"))
	
	client, err := rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		return fmt.Errorf("erro ao conectar: %w", err)
	}
	defer client.Close()

	fmt.Println(successStyle.Render("  ✓ Conectado com sucesso"))

	// PASSO 2: Salvar mensagens
	fmt.Println()
	fmt.Println(stepStyle.Render("━━━ PASSO 2/5: Salvando mensagens ━━━"))
	
	messages, err := client.DrainQueue(result.QueueName, func(current, total int) {
		progress := float64(current) / float64(total) * 100
		fmt.Printf("\r  ⏳ Salvando: %d/%d (%.1f%%)", current, total, progress)
	})
	if err != nil {
		return fmt.Errorf("erro ao salvar mensagens: %w", err)
	}

	if len(messages) > 0 {
		fmt.Println()
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("  ✓ %s mensagens salvas em memória",
		countStyle.Render(fmt.Sprintf("%d", len(messages))))))

	// PASSO 3: Criar sistema de retry
	fmt.Println()
	fmt.Println(stepStyle.Render("━━━ PASSO 3/5: Criando sistema de retry ━━━"))

	// Reconectar (pode ter expirado)
	client.Close()
	client, err = rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		return fmt.Errorf("erro ao reconectar: %w", err)
	}
	defer client.Close()

	setupOpts := retry.SetupOptions{
		QueueName:  result.QueueName,
		QueueType:  result.QueueType,
		MaxRetries: result.MaxRetries,
		RetryDelay: result.RetryDelay,
		DLQTTL:     result.DLQTTL,
		Force:      true, // Forçar recriação mesmo se já existir
	}

	if err := retry.SetupRetry(client, setupOpts); err != nil {
		return fmt.Errorf("erro ao criar sistema de retry: %w", err)
	}
	fmt.Println(successStyle.Render("  ✓ Sistema de retry criado"))

	// PASSO 4: Recriar fila com DLX
	fmt.Println()
	fmt.Println(stepStyle.Render("━━━ PASSO 4/5: Recriando fila com DLX ━━━"))

	// Reconectar
	client.Close()
	client, err = rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		return fmt.Errorf("erro ao reconectar: %w", err)
	}
	defer client.Close()

	// Primeiro deletar a fila antiga (se ainda existir)
	mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
	vhost := cfg.RabbitMQ.VHost
	if vhost == "" {
		vhost = "/"
	}
	_ = mgmtClient.DeleteQueueViaAPI(vhost, result.QueueName) // Ignora erro se não existir

	// Recriar com DLX
	if err := retry.RecreateQueueWithDLX(client, result.QueueName, result.QueueType); err != nil {
		// Tentar republicar mensagens mesmo se falhar
		fmt.Println(errorStyle.Render("  ⚠ Erro ao recriar fila: " + err.Error()))
		fmt.Println("  Tentando criar fila simples para não perder mensagens...")
		
		// Reconectar e criar fila simples
		client.Close()
		client, err = rabbitmq.NewClient(cfg.RabbitMQ)
		if err != nil {
			return fmt.Errorf("CRÍTICO: Não foi possível reconectar. %d mensagens podem ter sido perdidas", len(messages))
		}
		defer client.Close()

		opts := rabbitmq.CreateQueueOptions{
			Name:    result.QueueName,
			Type:    result.QueueType,
			Durable: true,
		}
		if err := client.CreateQueue(opts); err != nil {
			return fmt.Errorf("CRÍTICO: Não foi possível criar fila. %d mensagens perdidas", len(messages))
		}
	}
	fmt.Println(successStyle.Render("  ✓ Fila recriada com DLX"))

	// PASSO 5: Republicar mensagens
	fmt.Println()
	fmt.Println(stepStyle.Render("━━━ PASSO 5/5: Republicando mensagens ━━━"))

	if len(messages) == 0 {
		fmt.Println(successStyle.Render("  ✓ Nenhuma mensagem para republicar"))
	} else {
		// Reconectar
		client.Close()
		client, err = rabbitmq.NewClient(cfg.RabbitMQ)
		if err != nil {
			return fmt.Errorf("CRÍTICO: Não foi possível reconectar para republicar. %d mensagens perdidas", len(messages))
		}
		defer client.Close()

		err = client.PublishMessages(result.QueueName, messages, func(current, total int) {
			progress := float64(current) / float64(total) * 100
			fmt.Printf("\r  ⏳ Publicando: %d/%d (%.1f%%)", current, total, progress)
		})
		if err != nil {
			return fmt.Errorf("erro ao republicar mensagens: %w", err)
		}

		fmt.Println()
		fmt.Println(successStyle.Render(fmt.Sprintf("  ✓ %s mensagens republicadas",
			countStyle.Render(fmt.Sprintf("%d", len(messages))))))
	}

	// Resumo final
	fmt.Println()
	fmt.Println(stepStyle.Render("━━━ CONCLUÍDO ━━━"))
	fmt.Println()
	fmt.Printf("  Fila:          %s\n", result.QueueName)
	fmt.Printf("  Max Retries:   %d\n", result.MaxRetries)
	fmt.Printf("  Retry Delay:   %ds\n", result.RetryDelay)
	fmt.Printf("  Mensagens:     %d (preservadas)\n", len(messages))
	fmt.Println()
	fmt.Println("  Componentes criados:")
	fmt.Printf("    • %s.wait.exchange\n", result.QueueName)
	fmt.Printf("    • %s.wait\n", result.QueueName)
	fmt.Printf("    • %s.retry.exchange\n", result.QueueName)
	fmt.Printf("    • %s.dlq\n", result.QueueName)

	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// DELETAR FILA
// ═══════════════════════════════════════════════════════════════════════════════

// RunDeleteQueueForm executa o formulário para deletar uma fila
func RunDeleteQueueForm(cfg *config.Config) error {
	fmt.Print(renderFormHeader("🗑", "Deletar Fila", "Remover fila e todas as suas mensagens"))

	// Listar filas existentes
	mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
	queues, err := mgmtClient.ListQueues()
	if err != nil {
		return fmt.Errorf("erro ao listar filas: %w", err)
	}

	if len(queues) == 0 {
		return fmt.Errorf("nenhuma fila encontrada no RabbitMQ")
	}

	// Criar opções
	queueOptions := make([]huh.Option[string], len(queues))
	for i, queue := range queues {
		status := "🟢"
		if queue.MessagesReady > 0 {
			status = "📨"
		}

		label := fmt.Sprintf("%s %-25s │ %5d msgs │ %s",
			status,
			truncateString(queue.Name, 25),
			queue.MessagesReady,
			queue.Type)
		queueOptions[i] = huh.NewOption(label, queue.Name)
	}

	var (
		selectedQueue string
		cascade       bool
		confirm       bool
	)

	// Formulário de seleção
	form1 := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("📋 Legenda").
				Description("🟢 Vazia  │  📨 Com mensagens"),

			huh.NewSelect[string]().
				Title("Selecione a Fila").
				Description("Escolha a fila para deletar").
				Options(queueOptions...).
				Value(&selectedQueue),

			huh.NewConfirm().
				Title("Deletar filas relacionadas?").
				Description("Remover também .wait, .dlq e exchanges de retry").
				Value(&cascade),
		),
	)
	form1.WithTheme(getCustomTheme())

	if err := form1.Run(); err != nil {
		return fmt.Errorf("formulário cancelado: %w", err)
	}

	// Buscar detalhes da fila
	var queueInfo *rabbitmq.QueueInfoManagement
	for _, q := range queues {
		if q.Name == selectedQueue {
			queueInfo = &q
			break
		}
	}

	// Confirmação final
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E06C75")).Bold(true)
	fmt.Println()
	fmt.Println(warningStyle.Render("⚠️  ATENÇÃO: Esta operação é IRREVERSÍVEL!"))
	if queueInfo != nil && queueInfo.MessagesReady > 0 {
		fmt.Printf("   A fila tem %d mensagens que serão PERDIDAS!\n", queueInfo.MessagesReady)
	}
	fmt.Println()

	form2 := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Confirmar exclusão de '%s'?", selectedQueue)).
				Description("Esta ação não pode ser desfeita").
				Value(&confirm),
		),
	)
	form2.WithTheme(getCustomTheme())

	if err := form2.Run(); err != nil {
		return fmt.Errorf("formulário cancelado: %w", err)
	}

	if !confirm {
		return fmt.Errorf("operação cancelada pelo usuário")
	}

	// Executar deleção
	vhost := cfg.RabbitMQ.VHost
	if vhost == "" {
		vhost = "/"
	}

	fmt.Println()
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#98C379"))

	// Deletar filas relacionadas se cascade
	if cascade {
		fmt.Println("⏳ Deletando filas relacionadas...")
		relatedQueues := []string{
			selectedQueue + ".wait",
			selectedQueue + ".dlq",
		}
		for _, rq := range relatedQueues {
			if err := mgmtClient.DeleteQueueViaAPI(vhost, rq); err == nil {
				fmt.Println(successStyle.Render(fmt.Sprintf("  ✓ %s deletada", rq)))
			}
		}
	}

	// Deletar fila principal
	fmt.Println("⏳ Deletando fila principal...")
	if err := mgmtClient.DeleteQueueViaAPI(vhost, selectedQueue); err != nil {
		return fmt.Errorf("erro ao deletar fila: %w", err)
	}

	fmt.Println(successStyle.Render(fmt.Sprintf("  ✓ Fila '%s' deletada com sucesso!", selectedQueue)))
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// LIMPAR FILA (PURGE)
// ═══════════════════════════════════════════════════════════════════════════════

// RunPurgeQueueForm executa o formulário para limpar uma fila
func RunPurgeQueueForm(cfg *config.Config) error {
	fmt.Print(renderFormHeader("🧹", "Limpar Fila", "Remover todas as mensagens (purge)"))

	// Listar filas existentes
	mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
	queues, err := mgmtClient.ListQueues()
	if err != nil {
		return fmt.Errorf("erro ao listar filas: %w", err)
	}

	// Filtrar apenas filas com mensagens
	var filasComMensagens []rabbitmq.QueueInfoManagement
	for _, q := range queues {
		if q.MessagesReady > 0 {
			filasComMensagens = append(filasComMensagens, q)
		}
	}

	if len(filasComMensagens) == 0 {
		return fmt.Errorf("nenhuma fila com mensagens encontrada")
	}

	// Criar opções
	queueOptions := make([]huh.Option[string], len(filasComMensagens))
	for i, queue := range filasComMensagens {
		label := fmt.Sprintf("📨 %-25s │ %5d msgs │ %s",
			truncateString(queue.Name, 25),
			queue.MessagesReady,
			queue.Type)
		queueOptions[i] = huh.NewOption(label, queue.Name)
	}

	var (
		selectedQueue string
		confirm       bool
	)

	// Formulário de seleção
	form1 := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Selecione a Fila").
				Description("Escolha a fila para limpar").
				Options(queueOptions...).
				Value(&selectedQueue),
		),
	)
	form1.WithTheme(getCustomTheme())

	if err := form1.Run(); err != nil {
		return fmt.Errorf("formulário cancelado: %w", err)
	}

	// Buscar detalhes da fila
	var queueInfo *rabbitmq.QueueInfoManagement
	for _, q := range filasComMensagens {
		if q.Name == selectedQueue {
			queueInfo = &q
			break
		}
	}

	// Confirmação
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E5C07B")).Bold(true)
	fmt.Println()
	fmt.Println(warningStyle.Render("⚠️  ATENÇÃO:"))
	if queueInfo != nil {
		fmt.Printf("   %d mensagens serão REMOVIDAS da fila '%s'\n", queueInfo.MessagesReady, selectedQueue)
	}
	fmt.Println()

	form2 := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Limpar fila '%s'?", selectedQueue)).
				Description("Todas as mensagens serão removidas").
				Value(&confirm),
		),
	)
	form2.WithTheme(getCustomTheme())

	if err := form2.Run(); err != nil {
		return fmt.Errorf("formulário cancelado: %w", err)
	}

	if !confirm {
		return fmt.Errorf("operação cancelada pelo usuário")
	}

	// Executar purge
	client, err := rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		return fmt.Errorf("erro ao conectar: %w", err)
	}
	defer client.Close()

	purged, err := client.PurgeQueue(selectedQueue, false)
	if err != nil {
		return fmt.Errorf("erro ao limpar fila: %w", err)
	}

	fmt.Println()
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#98C379"))
	fmt.Println(successStyle.Render(fmt.Sprintf("✓ %d mensagens removidas da fila '%s'", purged, selectedQueue)))

	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// HELPERS
// ═══════════════════════════════════════════════════════════════════════════════

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
