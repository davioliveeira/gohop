package commands

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/davioliveeira/gohop/internal/config"
	"github.com/davioliveeira/gohop/internal/rabbitmq"
	"github.com/davioliveeira/gohop/internal/ui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Gerenciar configuração da CLI",
	Long:  "Comandos para configurar conexão com RabbitMQ e outras opções",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Configuração inicial interativa",
	Long:  "Inicia um formulário interativo para configurar a conexão com RabbitMQ",
	RunE:  runConfigInit,
}

var configTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Testa conexão com RabbitMQ",
	Long:  "Verifica se a configuração atual permite conectar ao RabbitMQ",
	RunE:  runConfigTest,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista perfis de configuração",
	Long:  "Mostra todos os perfis de configuração disponíveis",
	RunE:  runConfigList,
}

var configViewCmd = &cobra.Command{
	Use:   "view",
	Short: "Visualiza configuração atual",
	Long:  "Mostra a configuração atual carregada",
	RunE:  runConfigView,
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configTestCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configViewCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	// Header bonito
	fmt.Print(ui.SubMenuHeader("⚙", "Configurar Conexão RabbitMQ", "Configure a conexão com seu servidor RabbitMQ"))

	var (
		configMode     string = "simple"
		urlStr         string
		host           string
		port           string
		managementPort string
		username       string
		password       string
		vhost          string
		useTLS         bool
		profileName    string
		maxRetries     string
		retryDelay     string
		dlqTTL         string
	)

	theme := ui.GetCharmTheme()

	// Passo 1: Modo de configuração
	fmt.Println(ui.SubMenuInfo("Passo 1/3: Escolha o modo de configuração"))
	fmt.Println()

	modeForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("📋 Modo de Configuração").
				Description("Como deseja configurar a conexão?").
				Options(
					huh.NewOption("🔗 URL Simples - Apenas URL e credenciais", "simple"),
					huh.NewOption("⚙️  Completo - Todos os campos individualmente", "full"),
				).
				Value(&configMode),
		),
	)
	modeForm.WithTheme(theme)

	if err := modeForm.Run(); err != nil {
		return fmt.Errorf("cancelado: %w", err)
	}

	// Passo 2: Dados de conexão
	fmt.Println()
	fmt.Println(ui.SubMenuInfo("Passo 2/3: Configure os dados de conexão"))
	fmt.Println()

	var connectionForm *huh.Form

	if configMode == "simple" {
		connectionForm = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("🔗 URL de Conexão").
					Description("Formato: amqp://host:port/vhost ou amqps://host:port/vhost").
					Value(&urlStr).
					Placeholder("amqp://localhost:5672/").
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("URL é obrigatória")
						}
						_, err := urlParseConnectionURL(s)
						return err
					}),

				huh.NewInput().
					Title("👤 Username").
					Description("Usuário para autenticação no RabbitMQ").
					Value(&username).
					Placeholder("guest"),

				huh.NewInput().
					Title("🔒 Password").
					Description("Senha para autenticação").
					Value(&password).
					Password(true).
					Placeholder("••••••••"),
			),
		)
	} else {
		connectionForm = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("🖥️  Host").
					Description("Endereço do servidor RabbitMQ").
					Value(&host).
					Placeholder("localhost").
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("host é obrigatório")
						}
						return nil
					}),

				huh.NewInput().
					Title("🔌 Porta AMQP").
					Description("Porta para conexão AMQP (padrão: 5672, TLS: 5671)").
					Value(&port).
					Placeholder("5672"),

				huh.NewInput().
					Title("🌐 Porta Management API").
					Description("Porta para API de gerenciamento (padrão: 15672)").
					Value(&managementPort).
					Placeholder("15672"),
			),
			huh.NewGroup(
				huh.NewInput().
					Title("👤 Username").
					Description("Usuário para autenticação").
					Value(&username).
					Placeholder("guest"),

				huh.NewInput().
					Title("🔒 Password").
					Description("Senha para autenticação").
					Value(&password).
					Password(true).
					Placeholder("••••••••"),

				huh.NewInput().
					Title("📁 Virtual Host").
					Description("Virtual host do RabbitMQ (padrão: /)").
					Value(&vhost).
					Placeholder("/"),

				huh.NewConfirm().
					Title("🔐 Usar TLS/SSL?").
					Description("Habilitar conexão segura").
					Value(&useTLS),
			),
		)
	}
	connectionForm.WithTheme(theme)

	if err := connectionForm.Run(); err != nil {
		return fmt.Errorf("cancelado: %w", err)
	}

	// Passo 3: Configurações de retry
	fmt.Println()
	fmt.Println(ui.SubMenuInfo("Passo 3/3: Configure o sistema de retry (opcional)"))
	fmt.Println()

	retryForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("🔄 Máximo de Tentativas").
				Description("Quantas vezes tentar antes de enviar para DLQ").
				Value(&maxRetries).
				Placeholder("3"),

			huh.NewInput().
				Title("⏱️  Delay entre Tentativas (segundos)").
				Description("Tempo de espera antes de cada retry").
				Value(&retryDelay).
				Placeholder("5"),

			huh.NewInput().
				Title("📅 TTL da DLQ (milissegundos)").
				Description("Tempo de retenção na Dead Letter Queue (padrão: 7 dias)").
				Value(&dlqTTL).
				Placeholder("604800000"),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("📝 Nome do Perfil (opcional)").
				Description("Salvar como perfil específico (ex: dev, prod)").
				Value(&profileName).
				Placeholder("deixe vazio para usar 'default'"),
		),
	)
	retryForm.WithTheme(theme)

	if err := retryForm.Run(); err != nil {
		return fmt.Errorf("cancelado: %w", err)
	}

	// Processar dados
	if configMode == "simple" {
		parsedURL, err := urlParseConnectionURL(urlStr)
		if err != nil {
			return fmt.Errorf("erro ao processar URL: %w", err)
		}
		host = parsedURL.Host
		port = parsedURL.Port
		useTLS = parsedURL.UseTLS
		vhost = parsedURL.VHost

		// Porta management padrão
		if p, err := strconv.Atoi(parsedURL.Port); err == nil {
			if p == 5672 {
				managementPort = "15672"
			} else if p == 5671 {
				managementPort = "15671"
			} else {
				managementPort = strconv.Itoa(p + 10000)
			}
		}
	}

	// Aplicar valores padrão
	portInt := 5672
	if port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			portInt = p
		}
	}

	mgmtPortInt := 15672
	if managementPort != "" {
		if p, err := strconv.Atoi(managementPort); err == nil {
			mgmtPortInt = p
		}
	}

	if username == "" {
		username = "guest"
	}
	if password == "" {
		password = "guest"
	}
	if vhost == "" {
		vhost = "/"
	}

	maxRetriesInt := 3
	if maxRetries != "" {
		if r, err := strconv.Atoi(maxRetries); err == nil {
			maxRetriesInt = r
		}
	}

	retryDelayInt := 5
	if retryDelay != "" {
		if d, err := strconv.Atoi(retryDelay); err == nil {
			retryDelayInt = d
		}
	}

	dlqTTLInt := 604800000
	if dlqTTL != "" {
		if t, err := strconv.Atoi(dlqTTL); err == nil {
			dlqTTLInt = t
		}
	}

	// Criar configuração
	cfg := &config.Config{
		RabbitMQ: config.RabbitMQConfig{
			Host:           host,
			Port:           portInt,
			ManagementPort: mgmtPortInt,
			Username:       username,
			Password:       password,
			VHost:          vhost,
			UseTLS:         useTLS,
		},
		Retry: config.RetryConfig{
			MaxRetries: maxRetriesInt,
			RetryDelay: retryDelayInt,
			DLQTTL:     dlqTTLInt,
		},
	}

	// Salvar
	fmt.Println()
	fmt.Println(ui.SubMenuLoading("Salvando configuração"))

	if err := config.Save(cfg, profileName); err != nil {
		fmt.Println(ui.SubMenuError("Erro ao salvar configuração"))
		return fmt.Errorf("erro ao salvar: %w", err)
	}

	configPath := config.GetConfigDir()
	if profileName != "" {
		configPath = fmt.Sprintf("%s/config.%s.yaml", configPath, profileName)
	} else {
		configPath = fmt.Sprintf("%s/config.yaml", configPath)
	}

	fmt.Println(ui.SubMenuDone("Configuração salva!"))
	fmt.Println()

	// Resumo da configuração
	fmt.Println(ui.SubMenuSection("📋", "Resumo da Configuração"))
	fmt.Print(ui.SubMenuKeyValue("Host:", fmt.Sprintf("%s:%d", host, portInt), false))
	fmt.Print(ui.SubMenuKeyValue("Virtual Host:", vhost, false))
	fmt.Print(ui.SubMenuKeyValue("Username:", username, false))
	fmt.Print(ui.SubMenuKeyValue("TLS:", fmt.Sprintf("%v", useTLS), false))
	fmt.Print(ui.SubMenuKeyValue("Max Retries:", fmt.Sprintf("%d", maxRetriesInt), false))
	fmt.Print(ui.SubMenuKeyValue("Arquivo:", configPath, true))
	fmt.Println()

	// Testar conexão?
	var testConnection bool
	testForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("🔌 Testar conexão agora?").
				Description("Verificar se a configuração está correta").
				Value(&testConnection),
		),
	)
	testForm.WithTheme(theme)

	if err := testForm.Run(); err == nil && testConnection {
		return runConfigTest(cmd, args)
	}

	return nil
}

func runConfigTest(cmd *cobra.Command, args []string) error {
	fmt.Print(ui.SubMenuHeader("🔌", "Testar Conexão", "Verificando conexão com RabbitMQ"))

	cfg, err := config.Load(profile)
	if err != nil {
		fmt.Println(ui.SubMenuError("Erro ao carregar configuração"))
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		fmt.Println(ui.SubMenuError("Configuração inválida"))
		return fmt.Errorf("configuração inválida: %w", err)
	}

	// Mostrar dados
	fmt.Print(ui.SubMenuKeyValue("Host:", fmt.Sprintf("%s:%d", cfg.RabbitMQ.Host, cfg.RabbitMQ.Port), false))
	fmt.Print(ui.SubMenuKeyValue("Virtual Host:", cfg.RabbitMQ.VHost, false))
	fmt.Print(ui.SubMenuKeyValue("Username:", cfg.RabbitMQ.Username, false))
	fmt.Println()

	fmt.Println(ui.SubMenuLoading("Conectando ao RabbitMQ"))

	client, err := rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		fmt.Println(ui.SubMenuError("Falha na conexão"))
		return fmt.Errorf("erro ao conectar: %w", err)
	}
	defer client.Close()

	fmt.Println(ui.SubMenuDone("Conexão estabelecida com sucesso!"))
	fmt.Println()

	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	fmt.Print(ui.SubMenuHeader("📋", "Perfis de Configuração", "Lista de perfis disponíveis"))

	configDir := config.GetConfigDir()
	fmt.Print(ui.SubMenuKeyValue("Diretório:", configDir, false))
	fmt.Println()

	entries, err := os.ReadDir(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println(ui.SubMenuWarning("Nenhum perfil configurado ainda"))
			fmt.Println(ui.SubMenuHelp("Execute 'gohop config init' para criar um perfil"))
			return nil
		}
		return fmt.Errorf("erro ao ler diretório: %w", err)
	}

	var profiles []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			continue
		}

		if name == "config.yaml" {
			profiles = append(profiles, "default")
		} else if len(name) > 11 && name[:7] == "config." && name[len(name)-5:] == ".yaml" {
			profileName := name[7 : len(name)-5]
			profiles = append(profiles, profileName)
		}
	}

	if len(profiles) == 0 {
		fmt.Println(ui.SubMenuWarning("Nenhum perfil encontrado"))
		return nil
	}

	fmt.Println(ui.SubMenuSection("📁", "Perfis Disponíveis"))
	fmt.Print(ui.SubMenuList(profiles, "•"))

	fmt.Println(ui.SubMenuHelp("Use --profile <nome> para usar um perfil específico"))

	return nil
}

func runConfigView(cmd *cobra.Command, args []string) error {
	fmt.Print(ui.SubMenuHeader("📄", "Configuração Atual", "Detalhes da configuração carregada"))

	cfg, err := config.Load(profile)
	if err != nil {
		fmt.Println(ui.SubMenuError("Erro ao carregar configuração"))
		return fmt.Errorf("erro ao carregar: %w", err)
	}

	// RabbitMQ
	fmt.Println(ui.SubMenuSection("🐰", "RabbitMQ"))
	fmt.Print(ui.SubMenuKeyValue("Host:", cfg.RabbitMQ.Host, false))
	fmt.Print(ui.SubMenuKeyValue("Porta AMQP:", fmt.Sprintf("%d", cfg.RabbitMQ.Port), false))
	fmt.Print(ui.SubMenuKeyValue("Porta Management:", fmt.Sprintf("%d", cfg.RabbitMQ.ManagementPort), false))
	fmt.Print(ui.SubMenuKeyValue("Username:", cfg.RabbitMQ.Username, false))
	fmt.Print(ui.SubMenuKeyValue("Password:", maskPassword(cfg.RabbitMQ.Password), false))
	fmt.Print(ui.SubMenuKeyValue("Virtual Host:", cfg.RabbitMQ.VHost, false))
	fmt.Print(ui.SubMenuKeyValue("TLS:", fmt.Sprintf("%v", cfg.RabbitMQ.UseTLS), false))

	// Retry
	fmt.Println(ui.SubMenuSection("🔄", "Retry"))
	fmt.Print(ui.SubMenuKeyValue("Max Retries:", fmt.Sprintf("%d", cfg.Retry.MaxRetries), false))
	fmt.Print(ui.SubMenuKeyValue("Retry Delay:", fmt.Sprintf("%ds", cfg.Retry.RetryDelay), false))
	fmt.Print(ui.SubMenuKeyValue("DLQ TTL:", fmt.Sprintf("%dms", cfg.Retry.DLQTTL), false))

	fmt.Println()

	return nil
}

func maskPassword(pwd string) string {
	if len(pwd) == 0 {
		return "(não definida)"
	}
	if len(pwd) <= 2 {
		return "**"
	}
	return pwd[:1] + strings.Repeat("*", len(pwd)-2) + pwd[len(pwd)-1:]
}

type parsedURL struct {
	Host   string
	Port   string
	VHost  string
	UseTLS bool
}

func urlParseConnectionURL(urlStr string) (*parsedURL, error) {
	if !strings.HasPrefix(urlStr, "amqp://") && !strings.HasPrefix(urlStr, "amqps://") {
		urlStr = "amqp://" + urlStr
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("URL inválida: %w", err)
	}

	useTLS := parsed.Scheme == "amqps"
	host := parsed.Hostname()
	portStr := parsed.Port()

	if portStr == "" {
		if useTLS {
			portStr = "5671"
		} else {
			portStr = "5672"
		}
	}

	vhost := parsed.Path
	if vhost == "" || vhost == "/" {
		vhost = "/"
	} else {
		if vhost[0] == '/' {
			vhost = "/" + strings.TrimPrefix(vhost, "/")
		} else {
			vhost = "/" + vhost
		}
	}

	return &parsedURL{
		Host:   host,
		Port:   portStr,
		VHost:  vhost,
		UseTLS: useTLS,
	}, nil
}
