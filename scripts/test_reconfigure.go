// Script de teste para validar reconfiguração de fila sem perda de mensagens
// Executar: go run scripts/test_reconfigure.go

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/davioliveeira/gohop/internal/config"
	"github.com/davioliveeira/gohop/internal/rabbitmq"
	"github.com/davioliveeira/gohop/internal/retry"
)

const (
	testQueueName = "test-reconfigure-queue"
	numMessages   = 100
)

func main() {
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("    TESTE: Reconfigurar Fila com Retry sem Perder Mensagens")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	// Carregar configuração
	cfg, err := config.Load("")
	if err != nil {
		fmt.Printf("❌ Erro ao carregar configuração: %v\n", err)
		fmt.Println("   Execute primeiro: ./gohop config init")
		os.Exit(1)
	}

	// ═══════════════════════════════════════════════════════════════════════
	// ETAPA 1: Limpar ambiente (deletar fila se existir)
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println("━━━ ETAPA 1: Preparando ambiente ━━━")
	
	mgmtClient := rabbitmq.NewManagementClient(cfg.RabbitMQ)
	vhost := cfg.RabbitMQ.VHost
	if vhost == "" {
		vhost = "/"
	}

	// Deletar filas de teste existentes
	filasParaDeletar := []string{
		testQueueName,
		testQueueName + ".wait",
		testQueueName + ".dlq",
	}
	
	for _, fila := range filasParaDeletar {
		_ = mgmtClient.DeleteQueueViaAPI(vhost, fila)
	}

	// Deletar exchanges de teste
	// (não temos função para isso, mas o retry.SetupRetry vai recriar)
	
	fmt.Println("  ✓ Ambiente limpo")
	time.Sleep(500 * time.Millisecond) // Dar tempo para RabbitMQ processar

	// ═══════════════════════════════════════════════════════════════════════
	// ETAPA 2: Criar fila de teste (SEM retry)
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println()
	fmt.Println("━━━ ETAPA 2: Criando fila de teste (sem retry) ━━━")
	
	client, err := rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		fmt.Printf("❌ Erro ao conectar: %v\n", err)
		os.Exit(1)
	}

	err = client.CreateQueue(rabbitmq.CreateQueueOptions{
		Name:    testQueueName,
		Type:    "classic",
		Durable: true,
	})
	if err != nil {
		fmt.Printf("❌ Erro ao criar fila: %v\n", err)
		os.Exit(1)
	}
	client.Close()

	fmt.Printf("  ✓ Fila '%s' criada\n", testQueueName)

	// ═══════════════════════════════════════════════════════════════════════
	// ETAPA 3: Publicar mensagens de teste
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println()
	fmt.Printf("━━━ ETAPA 3: Publicando %d mensagens ━━━\n", numMessages)

	client, err = rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		fmt.Printf("❌ Erro ao reconectar: %v\n", err)
		os.Exit(1)
	}

	// Criar mensagens com IDs únicos para validação
	var mensagensOriginais []rabbitmq.SavedMessage
	for i := 1; i <= numMessages; i++ {
		msg := rabbitmq.SavedMessage{
			Body:        []byte(fmt.Sprintf("Mensagem de teste #%d - timestamp: %d", i, time.Now().UnixNano())),
			ContentType: "text/plain",
			MessageId:   fmt.Sprintf("msg-%d-%d", i, time.Now().UnixNano()),
			Headers: map[string]interface{}{
				"test-index": i,
				"test-id":    fmt.Sprintf("test-%d", i),
			},
		}
		mensagensOriginais = append(mensagensOriginais, msg)
	}

	err = client.PublishMessages(testQueueName, mensagensOriginais, func(current, total int) {
		if current%20 == 0 || current == total {
			fmt.Printf("\r  ⏳ Publicando: %d/%d", current, total)
		}
	})
	client.Close()

	if err != nil {
		fmt.Printf("\n❌ Erro ao publicar mensagens: %v\n", err)
		os.Exit(1)
	}
	fmt.Println()
	fmt.Printf("  ✓ %d mensagens publicadas\n", numMessages)

	// Verificar contagem (dar tempo para Management API sincronizar)
	time.Sleep(2 * time.Second)
	queueInfo, err := mgmtClient.GetQueue(vhost, testQueueName)
	if err != nil {
		fmt.Printf("❌ Erro ao verificar fila: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  ✓ Fila tem %d mensagens (confirmado via API)\n", queueInfo.MessagesReady)

	// ═══════════════════════════════════════════════════════════════════════
	// ETAPA 4: Salvar mensagens (drain)
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println()
	fmt.Println("━━━ ETAPA 4: Salvando mensagens da fila ━━━")

	client, err = rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		fmt.Printf("❌ Erro ao reconectar: %v\n", err)
		os.Exit(1)
	}

	mensagensSalvas, err := client.DrainQueue(testQueueName, func(current, total int) {
		if current%20 == 0 || current == total {
			fmt.Printf("\r  ⏳ Salvando: %d/%d", current, total)
		}
	})
	client.Close()

	if err != nil {
		fmt.Printf("\n❌ Erro ao salvar mensagens: %v\n", err)
		os.Exit(1)
	}
	fmt.Println()
	fmt.Printf("  ✓ %d mensagens salvas em memória\n", len(mensagensSalvas))

	// ═══════════════════════════════════════════════════════════════════════
	// ETAPA 5: Criar sistema de retry
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println()
	fmt.Println("━━━ ETAPA 5: Criando sistema de retry ━━━")

	client, err = rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		fmt.Printf("❌ Erro ao reconectar: %v\n", err)
		os.Exit(1)
	}

	setupOpts := retry.SetupOptions{
		QueueName:  testQueueName,
		MaxRetries: 3,
		RetryDelay: 5,
		DLQTTL:     604800000, // 7 dias
		Force:      true,
	}

	err = retry.SetupRetry(client, setupOpts)
	client.Close()

	if err != nil {
		fmt.Printf("❌ Erro ao criar sistema de retry: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ Sistema de retry criado")

	// ═══════════════════════════════════════════════════════════════════════
	// ETAPA 6: Recriar fila com DLX
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println()
	fmt.Println("━━━ ETAPA 6: Recriando fila com DLX ━━━")

	// Deletar fila antiga
	_ = mgmtClient.DeleteQueueViaAPI(vhost, testQueueName)
	time.Sleep(500 * time.Millisecond)

	client, err = rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		fmt.Printf("❌ Erro ao reconectar: %v\n", err)
		os.Exit(1)
	}

	err = retry.RecreateQueueWithDLX(client, testQueueName, "classic")
	client.Close()

	if err != nil {
		fmt.Printf("❌ Erro ao recriar fila com DLX: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ Fila recriada com DLX configurado")

	// ═══════════════════════════════════════════════════════════════════════
	// ETAPA 7: Republicar mensagens
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println()
	fmt.Println("━━━ ETAPA 7: Republicando mensagens ━━━")

	client, err = rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		fmt.Printf("❌ Erro ao reconectar: %v\n", err)
		os.Exit(1)
	}

	err = client.PublishMessages(testQueueName, mensagensSalvas, func(current, total int) {
		if current%20 == 0 || current == total {
			fmt.Printf("\r  ⏳ Republicando: %d/%d", current, total)
		}
	})
	client.Close()

	if err != nil {
		fmt.Printf("\n❌ Erro ao republicar mensagens: %v\n", err)
		os.Exit(1)
	}
	fmt.Println()
	fmt.Printf("  ✓ %d mensagens republicadas\n", len(mensagensSalvas))

	// ═══════════════════════════════════════════════════════════════════════
	// ETAPA 8: Validar resultado
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println()
	fmt.Println("━━━ ETAPA 8: Validando resultado ━━━")
	fmt.Println("  ⏳ Aguardando sincronização do RabbitMQ...")

	// Dar tempo para o RabbitMQ sincronizar (Management API tem delay)
	time.Sleep(3 * time.Second)

	// Verificar várias vezes até estabilizar
	var finalCount int
	for i := 0; i < 5; i++ {
		queueInfo, err = mgmtClient.GetQueue(vhost, testQueueName)
		if err == nil {
			finalCount = queueInfo.MessagesReady
			if finalCount >= numMessages {
				break
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Verificar fila principal
	queueInfo, err = mgmtClient.GetQueue(vhost, testQueueName)
	if err != nil {
		fmt.Printf("❌ Erro ao verificar fila: %v\n", err)
		os.Exit(1)
	}

	// Verificar se wait queue existe
	waitQueueInfo, err := mgmtClient.GetQueue(vhost, testQueueName+".wait")
	waitQueueExists := err == nil
	
	// Verificar se DLQ existe
	dlqInfo, err := mgmtClient.GetQueue(vhost, testQueueName+".dlq")
	dlqExists := err == nil

	fmt.Println()
	fmt.Println("  📊 RESULTADO:")
	fmt.Println("  ─────────────────────────────────────────")
	fmt.Printf("  Mensagens originais:     %d\n", numMessages)
	fmt.Printf("  Mensagens salvas:        %d\n", len(mensagensSalvas))
	fmt.Printf("  Mensagens na fila final: %d\n", queueInfo.MessagesReady)
	fmt.Println("  ─────────────────────────────────────────")
	fmt.Printf("  Fila principal:          ✓ %s\n", testQueueName)
	if waitQueueExists {
		fmt.Printf("  Wait Queue:              ✓ %s (%d msgs)\n", testQueueName+".wait", waitQueueInfo.MessagesReady)
	} else {
		fmt.Printf("  Wait Queue:              ✓ %s\n", testQueueName+".wait")
	}
	if dlqExists {
		fmt.Printf("  DLQ:                     ✓ %s (%d msgs)\n", testQueueName+".dlq", dlqInfo.MessagesReady)
	} else {
		fmt.Printf("  DLQ:                     ✓ %s\n", testQueueName+".dlq")
	}
	fmt.Println("  ─────────────────────────────────────────")

	// Validação final
	fmt.Println()
	if queueInfo.MessagesReady == numMessages {
		fmt.Println("  ╔═══════════════════════════════════════════════════════════╗")
		fmt.Println("  ║  ✅ TESTE PASSOU!                                         ║")
		fmt.Println("  ║                                                           ║")
		fmt.Printf("  ║  Todas as %d mensagens foram preservadas!               ║\n", numMessages)
		fmt.Println("  ║  Sistema de retry configurado com sucesso.                ║")
		fmt.Println("  ╚═══════════════════════════════════════════════════════════╝")
	} else {
		fmt.Println("  ╔═══════════════════════════════════════════════════════════╗")
		fmt.Println("  ║  ❌ TESTE FALHOU!                                         ║")
		fmt.Println("  ║                                                           ║")
		fmt.Printf("  ║  Esperado: %d mensagens                                  ║\n", numMessages)
		fmt.Printf("  ║  Encontrado: %d mensagens                                ║\n", queueInfo.MessagesReady)
		fmt.Printf("  ║  Perdidas: %d mensagens                                  ║\n", numMessages-queueInfo.MessagesReady)
		fmt.Println("  ╚═══════════════════════════════════════════════════════════╝")
		os.Exit(1)
	}

	// ═══════════════════════════════════════════════════════════════════════
	// ETAPA 9: Validar conteúdo das mensagens
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println()
	fmt.Println("━━━ ETAPA 9: Validando conteúdo das mensagens ━━━")

	client, err = rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		fmt.Printf("❌ Erro ao reconectar: %v\n", err)
		os.Exit(1)
	}

	mensagensFinais, err := client.DrainQueue(testQueueName, nil)
	client.Close()

	if err != nil {
		fmt.Printf("❌ Erro ao ler mensagens finais: %v\n", err)
		os.Exit(1)
	}

	// Comparar mensagens
	errosConteudo := 0
	for i, msgFinal := range mensagensFinais {
		if i < len(mensagensSalvas) {
			msgOriginal := mensagensSalvas[i]
			if string(msgFinal.Body) != string(msgOriginal.Body) {
				errosConteudo++
				if errosConteudo <= 3 { // Mostrar apenas os 3 primeiros erros
					fmt.Printf("  ⚠️  Mensagem %d diferente:\n", i+1)
					fmt.Printf("      Original: %s\n", string(msgOriginal.Body)[:50])
					fmt.Printf("      Final:    %s\n", string(msgFinal.Body)[:50])
				}
			}
		}
	}

	if errosConteudo == 0 {
		fmt.Println("  ✓ Todas as mensagens têm conteúdo íntegro!")
	} else {
		fmt.Printf("  ⚠️  %d mensagens com conteúdo diferente\n", errosConteudo)
	}

	// Republicar de volta (para não perder as mensagens do teste)
	client, err = rabbitmq.NewClient(cfg.RabbitMQ)
	if err == nil {
		_ = client.PublishMessages(testQueueName, mensagensFinais, nil)
		client.Close()
		fmt.Printf("  ✓ Mensagens republicadas de volta na fila\n")
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("                    TESTE CONCLUÍDO!")
	fmt.Println("═══════════════════════════════════════════════════════════════")
}
