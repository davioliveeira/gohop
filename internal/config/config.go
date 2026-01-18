package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config representa a configuração completa do RabbitMQ
type Config struct {
	RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
	Retry    RetryConfig    `mapstructure:"retry"`
}

// RabbitMQConfig contém as configurações de conexão
type RabbitMQConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	ManagementPort int    `mapstructure:"management_port"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password"`
	VHost           string `mapstructure:"vhost"`
	UseTLS          bool   `mapstructure:"use_tls"`
}

// RetryConfig contém as configurações padrão de retry
type RetryConfig struct {
	MaxRetries int `mapstructure:"max_retries"`
	RetryDelay int `mapstructure:"retry_delay"` // em segundos
	DLQTTL      int `mapstructure:"dlq_ttl"`      // em milissegundos
}

var (
	globalConfig *Config
	viperConfig  *viper.Viper
)

// Load carrega a configuração de arquivos e variáveis de ambiente
func Load(profileName string) (*Config, error) {
	viperConfig = viper.New()

	// Carregar .env se existir
	_ = godotenv.Load()

	// Configurar Viper
	viperConfig.SetConfigName("config")
	viperConfig.SetConfigType("yaml")
	viperConfig.AddConfigPath("$HOME/.gohop")
	viperConfig.AddConfigPath(".")

	// Variáveis de ambiente (têm precedência)
	viperConfig.SetEnvPrefix("RABBITMQ")
	viperConfig.AutomaticEnv()

	// Bindings de variáveis de ambiente
	viperConfig.BindEnv("rabbitmq.host", "RABBITMQ_HOST")
	viperConfig.BindEnv("rabbitmq.port", "RABBITMQ_PORT")
	viperConfig.BindEnv("rabbitmq.management_port", "RABBITMQ_MANAGEMENT_PORT")
	viperConfig.BindEnv("rabbitmq.username", "RABBITMQ_USER")
	viperConfig.BindEnv("rabbitmq.password", "RABBITMQ_PASSWORD")
	viperConfig.BindEnv("rabbitmq.vhost", "RABBITMQ_VHOST")
	viperConfig.BindEnv("rabbitmq.use_tls", "RABBITMQ_USE_TLS")

	viperConfig.BindEnv("retry.max_retries", "MAX_RETRIES")
	viperConfig.BindEnv("retry.retry_delay", "RETRY_DELAY")
	viperConfig.BindEnv("retry.dlq_ttl", "DLQ_MESSAGE_TTL")

	// Valores padrão
	setDefaults()

	// Se profile especificado, tentar carregar
	if profileName != "" {
		viperConfig.SetConfigName(fmt.Sprintf("config.%s", profileName))
	}

	// Tentar ler arquivo de configuração (não é erro se não existir)
	if err := viperConfig.ReadInConfig(); err != nil {
		// Se não encontrar arquivo, usar apenas defaults e env vars
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("erro ao ler arquivo de configuração: %w", err)
		}
	}

	// Unmarshal para struct
	var cfg Config
	if err := viperConfig.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("erro ao fazer parse da configuração: %w", err)
	}

	globalConfig = &cfg
	return &cfg, nil
}

// setDefaults define valores padrão
func setDefaults() {
	viperConfig.SetDefault("rabbitmq.host", "localhost")
	viperConfig.SetDefault("rabbitmq.port", 5672)
	viperConfig.SetDefault("rabbitmq.management_port", 15672)
	viperConfig.SetDefault("rabbitmq.username", "guest")
	viperConfig.SetDefault("rabbitmq.password", "guest")
	viperConfig.SetDefault("rabbitmq.vhost", "/")
	viperConfig.SetDefault("rabbitmq.use_tls", false)

	viperConfig.SetDefault("retry.max_retries", 3)
	viperConfig.SetDefault("retry.retry_delay", 5)
	viperConfig.SetDefault("retry.dlq_ttl", 604800000) // 7 dias em ms
}

// Save salva a configuração em arquivo YAML
func Save(cfg *Config, profileName string) error {
	configDir := filepath.Join(os.Getenv("HOME"), ".gohop")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("erro ao criar diretório de configuração: %w", err)
	}

	configFile := "config.yaml"
	if profileName != "" {
		configFile = fmt.Sprintf("config.%s.yaml", profileName)
	}

	configPath := filepath.Join(configDir, configFile)

	// Criar novo viper para salvar
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile(configPath)

	// Mapear struct para viper
	v.Set("rabbitmq", cfg.RabbitMQ)
	v.Set("retry", cfg.Retry)

	if err := v.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("erro ao salvar configuração: %w", err)
	}

	globalConfig = cfg
	return nil
}

// Get retorna a configuração global carregada
func Get() *Config {
	if globalConfig == nil {
		// Tentar carregar com defaults
		cfg, _ := Load("")
		return cfg
	}
	return globalConfig
}

// GetConfigDir retorna o diretório de configuração
func GetConfigDir() string {
	return filepath.Join(os.Getenv("HOME"), ".gohop")
}

// Validate valida se a configuração está completa
func (c *Config) Validate() error {
	if c.RabbitMQ.Host == "" {
		return fmt.Errorf("host não configurado")
	}
	if c.RabbitMQ.Port == 0 {
		return fmt.Errorf("porta não configurada")
	}
	if c.RabbitMQ.Username == "" {
		return fmt.Errorf("username não configurado")
	}
	if c.RabbitMQ.Password == "" {
		return fmt.Errorf("password não configurado")
	}
	return nil
}
