package commands

import (
	"fmt"
	"os"

	"github.com/davioliveeira/gohop/internal/config"
	"github.com/davioliveeira/gohop/internal/ui"
	"github.com/spf13/cobra"
)

var (
	// Flags globais
	verbose   bool
	noColor   bool
	outputFmt string
	profile   string
)

var rootCmd = &cobra.Command{
	Use:   "gohop",
	Short: "CLI moderna para gerenciar RabbitMQ com sistema de retry",
	Long: `Uma CLI moderna e interativa para gerenciar RabbitMQ, incluindo:
- Configuração interativa de conexão
- Criação automática de filas com sistema de retry
- Monitoramento em tempo real
- Gerenciamento completo de exchanges e bindings

Execute 'gohop' sem argumentos para abrir o menu interativo.`,
	Version: "0.1.0",
	RunE:    runRootMenu,
}

// Execute adiciona todos os comandos filhos ao root e executa
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Flags globais
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "habilitar logs detalhados")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "desabilitar cores na saída")
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "table", "formato de saída (table|json)")
	rootCmd.PersistentFlags().StringVarP(&profile, "profile", "p", "", "perfil de configuração a usar")

	// Comandos filhos
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(queueCmd)
	rootCmd.AddCommand(retryCmd)
	rootCmd.AddCommand(monitorCmd)

	// Customização do help será feita via Glamour (a implementar)
}

// runRootMenu executa o menu principal interativo quando não há subcomandos
func runRootMenu(cmd *cobra.Command, args []string) error {
	// Carregar configuração (pode não existir ainda)
	cfg, _ := config.Load(profile)

	// Executar menu interativo
	return ui.RunMainMenu(cfg)
}

// getLogger retorna um logger configurado baseado nas flags globais
func getLogger() {
	// TODO: Implementar logger com zerolog
}

// checkConfig verifica se a configuração existe antes de executar comandos
func checkConfig() error {
	configPath := os.Getenv("HOME") + "/.gohop/config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("configuração não encontrada. Execute 'gohop config init' primeiro")
	}
	return nil
}
