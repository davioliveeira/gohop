package main

import (
	"fmt"
	"os"
	"time"

	"github.com/davioliveeira/gohop/internal/config"
	"github.com/davioliveeira/gohop/internal/rabbitmq"
)

func main() {
	cfg, err := config.Load("")
	if err != nil {
		fmt.Printf("❌ Erro: %v\n", err)
		os.Exit(1)
	}

	// Criar fila
	client, err := rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		fmt.Printf("❌ Erro: %v\n", err)
		os.Exit(1)
	}

	err = client.CreateQueue(rabbitmq.CreateQueueOptions{
		Name:    "test-reconfigure-queue",
		Type:    "classic",
		Durable: true,
	})
	if err != nil {
		fmt.Printf("❌ Erro ao criar fila: %v\n", err)
		os.Exit(1)
	}
	client.Close()
	fmt.Println("✅ Fila 'test-reconfigure-queue' criada!")

	// Publicar 100 mensagens
	client, err = rabbitmq.NewClient(cfg.RabbitMQ)
	if err != nil {
		fmt.Printf("❌ Erro: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	var msgs []rabbitmq.SavedMessage
	for i := 1; i <= 100; i++ {
		msgs = append(msgs, rabbitmq.SavedMessage{
			Body:        []byte(fmt.Sprintf("Mensagem de teste #%d - %d", i, time.Now().UnixNano())),
			ContentType: "text/plain",
			MessageId:   fmt.Sprintf("msg-%d", i),
		})
	}

	err = client.PublishMessages("test-reconfigure-queue", msgs, func(c, t int) {
		if c%20 == 0 || c == t {
			fmt.Printf("\r⏳ Publicando: %d/%d", c, t)
		}
	})
	if err != nil {
		fmt.Printf("\n❌ Erro: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("\n✅ 100 mensagens publicadas com sucesso!")
}
