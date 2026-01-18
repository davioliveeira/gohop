# 🚀 Guia Rápido - Retry com 3 Tentativas

## ✅ Configuração já está pronta!

Seu `.env` já está configurado para **3 tentativas** antes de enviar para DLQ.

---

## 📝 Passo a Passo para Configurar uma Fila

### 1️⃣ Criar estrutura de retry para uma fila

```bash
python dlq_setup_with_retry.py --queue NOME_DA_FILA
```

Exemplo:
```bash
python dlq_setup_with_retry.py --queue cartpanda_physical
```

Isso cria:
- ✅ `cartpanda_physical.wait` (aguarda 5s antes de retry)
- ✅ `cartpanda_physical.wait.exchange`
- ✅ `cartpanda_physical.retry` (exchange de retry)
- ✅ `cartpanda_physical.dlq` (destino final após 3 falhas)

---

### 2️⃣ Deletar a fila principal

**ATENÇÃO:** Faça backup se houver mensagens importantes!

1. Acesse: http://SEU_HOST:15672
2. Vá em **Queues** → encontre `cartpanda_physical`
3. Clique em **Delete**

---

### 3️⃣ Recriar a fila com configuração de DLQ

```bash
python dlq_setup_with_retry.py --recreate cartpanda_physical
```

Isso recria a fila com `x-dead-letter-exchange` configurado.

---

## 🧪 Testar o Sistema

### Iniciar Consumer (Terminal 1)

```bash
python consumer_with_retry_example.py --queue cartpanda_physical
```

### Enviar Mensagem de Teste (Terminal 2)

**Mensagem que vai FALHAR e testar retry:**
```bash
python test_retry_publisher.py --queue cartpanda_physical --type fail
```

Você verá no consumer:
```
📨 Processing message (attempt 1/3)
   ❌ Error processing message
   🔄 Retrying...

[após 5 segundos]

📨 Processing message (attempt 2/3)
   ❌ Error processing message
   🔄 Retrying...

[após 5 segundos]

📨 Processing message (attempt 3/3)
   ❌ Error processing message
   ⚠️  Max retries reached. Sending to DLQ.
```

---

## 📊 Monitorar Estatísticas

### Ver estatísticas de uma fila:
```bash
python monitor_retry.py --queue cartpanda_physical
```

Mostra:
- Mensagens na fila principal
- Mensagens aguardando retry (wait queue)
- Mensagens na DLQ
- Número de tentativas de cada mensagem na DLQ

### Listar todas as filas com retry:
```bash
python monitor_retry.py --list
```

---

## 🎯 Fluxo de uma Mensagem que Falha

```
1. Mensagem chega → Consumer processa → FALHA (tentativa 1/3)
   └─> Vai para WAIT QUEUE (aguarda 5s)

2. Após 5s → Retry → Consumer processa → FALHA (tentativa 2/3)
   └─> Vai para WAIT QUEUE (aguarda 5s)

3. Após 5s → Retry → Consumer processa → FALHA (tentativa 3/3)
   └─> Vai para WAIT QUEUE (aguarda 5s)

4. Após 5s → MAX RETRIES atingido → Vai para DLQ ☠️
```

**Tempo total: ~15 segundos** (3x 5s de espera)

---

## ⚙️ Personalizar Configurações

### Alterar número de tentativas

Edite `.env`:
```bash
MAX_RETRIES=5  # 5 tentativas em vez de 3
```

### Alterar tempo entre tentativas

Edite `dlq_setup_with_retry.py` (linha 72):
```python
'x-message-ttl': 10000,  # 10 segundos
```

### Alterar tempo de retenção na DLQ

Edite `.env`:
```bash
DLQ_MESSAGE_TTL=1209600000  # 14 dias (em milissegundos)
```

---

## 📋 Checklist de Setup

- [ ] Rodar `python dlq_setup_with_retry.py --queue NOME_FILA`
- [ ] Deletar fila principal pela UI do RabbitMQ
- [ ] Rodar `python dlq_setup_with_retry.py --recreate NOME_FILA`
- [ ] Testar com `test_retry_publisher.py`
- [ ] Verificar na UI que as 4 filas foram criadas
- [ ] Monitorar com `monitor_retry.py`

---

## 🔗 Arquivos Criados

```
✅ dlq_setup_with_retry.py      → Setup do sistema de retry
✅ consumer_with_retry_example.py → Exemplo de consumer
✅ test_retry_publisher.py      → Publisher para testes
✅ monitor_retry.py             → Monitoramento de estatísticas
✅ README_RETRY_SETUP.md        → Documentação completa
✅ ARQUITETURA_RETRY.md         → Diagramas e explicação
✅ GUIA_RETRY_3X.md            → Este guia
```

---

## ❓ Dúvidas Comuns

**P: As mensagens vão para DLQ automaticamente após 3 falhas?**
R: Sim! O RabbitMQ rastreia o número de rejeições no header `x-death`.

**P: Posso mudar o delay entre tentativas?**
R: Sim, basta alterar o `x-message-ttl` na wait queue.

**P: Como ver quantas tentativas uma mensagem teve?**
R: Use `python monitor_retry.py --queue NOME_FILA` para ver as estatísticas.

**P: Mensagens na DLQ são deletadas automaticamente?**
R: Sim, após 7 dias (configurável via `DLQ_MESSAGE_TTL`).

---

## 🎉 Pronto!

Agora você tem um sistema completo de retry com 3 tentativas automáticas antes de enviar para DLQ!
