# RabbitMQ DLQ with Retry Logic Setup

Este guia mostra como configurar uma Dead Letter Queue (DLQ) com lógica de retry automático - mensagens que falharem serão tentadas **3 vezes** antes de irem para a DLQ final.

## 🏗️ Arquitetura

```
┌─────────────┐  reject   ┌──────────────┐  TTL 5s  ┌───────────────┐
│ Main Queue  │ ───────> │ Wait Queue   │ ──────> │ Retry Exchange│
│             │          │ (5s delay)   │         │               │
└─────────────┘          └──────────────┘         └───────┬───────┘
      ▲                                                    │
      │                                                    │
      │ retry < 3x                                         │ retry >= 3x
      │                                                    │
      └────────────────────────────────────────────────────┴──────────> DLQ
```

### Como funciona:

1. **Mensagem falha** → vai para **Wait Queue**
2. **Wait Queue** → aguarda **5 segundos** (TTL)
3. Após 5s → vai para **Retry Exchange**
4. **Retry Exchange** verifica o header `x-death`:
   - Se `tentativas < 3` → **volta para Main Queue**
   - Se `tentativas >= 3` → **vai para DLQ**

## 📋 Pré-requisitos

```bash
pip install pika requests python-dotenv tabulate
```

## ⚙️ Configuração

### 1. Configure as variáveis de ambiente (`.env`)

```bash
# RabbitMQ Connection
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_MANAGEMENT_PORT=15672
RABBITMQ_USER=guest
RABBITMQ_PASSWORD=guest
RABBITMQ_VHOST=/

# DLQ Settings
MAX_RETRIES=3
MESSAGE_TTL=86400000      # 24 hours (não usado no retry)
DLQ_MESSAGE_TTL=604800000 # 7 days (mensagens na DLQ)
```

### 2. Criar as filas e exchanges

```bash
# Dry-run para ver o que será criado
python dlq_setup_with_retry.py --queue cartpanda_physical --dry-run

# Criar as filas e exchanges de retry
python dlq_setup_with_retry.py --queue cartpanda_physical
```

Isso criará:
- ✅ `cartpanda_physical.wait` (fila de espera com TTL de 5s)
- ✅ `cartpanda_physical.wait.exchange` (recebe mensagens rejeitadas)
- ✅ `cartpanda_physical.retry` (exchange de retry)
- ✅ `cartpanda_physical.dlq` (fila final após 3 tentativas)

### 3. Recriar a fila principal

**IMPORTANTE:** Você precisa deletar a fila principal manualmente pela UI do RabbitMQ antes deste passo!

```bash
# 1. Pare todos os workflows do n8n que usam a fila
# 2. Delete a fila 'cartpanda_physical' pela UI do RabbitMQ
# 3. Recrie a fila com a configuração de DLQ

python dlq_setup_with_retry.py --recreate cartpanda_physical
```

Isso recriará a fila `cartpanda_physical` com `x-dead-letter-exchange` apontando para a wait queue.

## 🧪 Testando o Sistema

### 1. Iniciar o Consumer

Abra um terminal e rode:

```bash
python consumer_with_retry_example.py --queue cartpanda_physical
```

O consumer ficará escutando mensagens e processando com lógica de retry.

### 2. Publicar Mensagens de Teste

#### Mensagem que vai FALHAR (testará o retry):

```bash
python test_retry_publisher.py --queue cartpanda_physical --type fail
```

Esta mensagem vai:
1. **Tentativa 1** → falhar → ir para wait queue (5s) → voltar para main queue
2. **Tentativa 2** → falhar → ir para wait queue (5s) → voltar para main queue
3. **Tentativa 3** → falhar → ir para wait queue (5s) → voltar para main queue
4. **Após 3 tentativas** → **ir para DLQ** ❌

#### Mensagem que vai ter SUCESSO:

```bash
python test_retry_publisher.py --queue cartpanda_physical --type success
```

Esta mensagem será processada com sucesso na primeira tentativa. ✅

## 📊 Monitoramento

### Via RabbitMQ UI

Acesse: http://localhost:15672

Você verá as filas:
- `cartpanda_physical` (quorum, com DLX)
- `cartpanda_physical.wait` (classic, TTL 5s)
- `cartpanda_physical.dlq` (classic, TTL 7 dias)

### Verificar headers de retry

O RabbitMQ adiciona automaticamente o header `x-death` nas mensagens rejeitadas, que contém:
- `count`: número de vezes que a mensagem foi rejeitada
- `exchange`: exchange que recebeu a rejeição
- `queue`: fila que rejeitou
- `time`: timestamp da rejeição

## 🔧 Integração com n8n

Para usar isso no n8n, você precisa:

1. **RabbitMQ Trigger Node**: configurar para consumir da fila principal
2. **Error Workflow**: configurar para rejeitar mensagens em caso de erro
3. **Manual Retry**: usar `basic_reject(requeue=False)` para acionar o retry

### Exemplo de configuração no n8n:

```javascript
// No node de erro/catch
const channel = $input.item.json.channel;
const deliveryTag = $input.item.json.deliveryTag;

// Rejeitar sem requeue para ativar o DLX
channel.reject(deliveryTag, false);
```

## 🎯 Configurações Personalizadas

### Alterar número máximo de retries

Edite o arquivo `.env`:

```bash
MAX_RETRIES=5  # Tentar 5 vezes antes da DLQ
```

### Alterar tempo de espera entre retries

Edite `dlq_setup_with_retry.py` linha 72:

```python
'x-message-ttl': 10000,  # 10 segundos em vez de 5
```

### Alterar tempo de retenção na DLQ

Edite o arquivo `.env`:

```bash
DLQ_MESSAGE_TTL=1209600000  # 14 dias (em milissegundos)
```

## 🚨 Troubleshooting

### Mensagens não estão sendo retriadas

✅ Verifique se o consumer está usando `basic_reject(requeue=False)`
✅ Confirme que a fila principal tem `x-dead-letter-exchange` configurado
✅ Verifique se a wait queue tem TTL configurado

### Mensagens indo direto para DLQ

✅ Verifique o valor de `MAX_RETRIES` no `.env`
✅ Confirme que o consumer está lendo o header `x-death` corretamente

### Consumer não está processando

✅ Verifique se a fila existe e tem mensagens
✅ Confirme a conexão com RabbitMQ
✅ Verifique os logs do consumer

## 📚 Estrutura de Arquivos

```
rabbit/
├── config.py                          # Configurações
├── dlq_setup_with_retry.py           # Setup do sistema de retry
├── consumer_with_retry_example.py    # Exemplo de consumer
├── test_retry_publisher.py           # Publisher de teste
├── .env                               # Variáveis de ambiente
└── README_RETRY_SETUP.md             # Este arquivo
```

## 🔗 Referências

- [RabbitMQ Dead Letter Exchanges](https://www.rabbitmq.com/dlx.html)
- [RabbitMQ TTL](https://www.rabbitmq.com/ttl.html)
- [RabbitMQ Headers Exchange](https://www.rabbitmq.com/tutorials/amqp-concepts.html#exchange-headers)
