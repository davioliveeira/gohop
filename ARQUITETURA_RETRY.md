# Arquitetura de Retry com RabbitMQ

## 🎯 Fluxo Completo

```
                                    ┌─────────────────────────────────────────┐
                                    │         CONSUMIDOR (n8n/Python)         │
                                    │  - Processa mensagem                    │
                                    │  - Em caso de erro: basic_reject()      │
                                    └──────────────┬──────────────────────────┘
                                                   │
                                                   ▼
                                    ┌──────────────────────────────┐
                                    │   📬 MAIN QUEUE (quorum)     │
                                    │   cartpanda_physical         │
                                    │                              │
                                    │   DLX: .wait.exchange        │
                                    └──────────────┬───────────────┘
                                                   │
                         ┌─────────────────────────┘
                         │ reject (requeue=False)
                         ▼
          ┌──────────────────────────────┐
          │  ⏳ WAIT EXCHANGE (fanout)   │
          │  cartpanda_physical.wait.ex  │
          └──────────────┬───────────────┘
                         │
                         ▼
          ┌──────────────────────────────┐
          │   💤 WAIT QUEUE (classic)    │
          │   cartpanda_physical.wait    │
          │                              │
          │   TTL: 5 segundos            │
          │   DLX: .retry exchange       │
          └──────────────┬───────────────┘
                         │
                         │ (aguarda 5s)
                         │
                         ▼
          ┌──────────────────────────────┐
          │  🔄 RETRY EXCHANGE (fanout)  │
          │  cartpanda_physical.retry    │
          │                              │
          │  Analisa header x-death      │
          └──────────┬──────────┬────────┘
                     │          │
        ┌────────────┘          └──────────────┐
        │                                      │
        │ tentativas < 3                       │ tentativas >= 3
        ▼                                      ▼
┌───────────────────┐              ┌──────────────────────┐
│   🔁 VOLTA PARA   │              │   ☠️ DLQ (classic)   │
│    MAIN QUEUE     │              │  cartpanda_physical  │
│                   │              │        .dlq          │
│  (tenta de novo)  │              │                      │
└───────────────────┘              │  TTL: 7 dias         │
                                   └──────────────────────┘
```

## 📊 Exemplo de Mensagem com 3 Tentativas

### Tentativa 1 (headers iniciais)
```json
{
  "data": "minha mensagem",
  "headers": {}
}
```
❌ **FALHOU** → vai para wait queue

---

### Tentativa 2 (após 5 segundos)
```json
{
  "data": "minha mensagem",
  "headers": {
    "x-death": [{
      "count": 1,
      "queue": "cartpanda_physical",
      "exchange": "cartpanda_physical.wait.exchange",
      "time": "2025-01-16T10:30:00Z"
    }]
  }
}
```
❌ **FALHOU** → vai para wait queue novamente

---

### Tentativa 3 (após mais 5 segundos)
```json
{
  "data": "minha mensagem",
  "headers": {
    "x-death": [{
      "count": 2,
      "queue": "cartpanda_physical",
      "exchange": "cartpanda_physical.wait.exchange",
      "time": "2025-01-16T10:30:05Z"
    }]
  }
}
```
❌ **FALHOU** → vai para wait queue novamente

---

### Após 3 Tentativas (vai para DLQ)
```json
{
  "data": "minha mensagem",
  "headers": {
    "x-death": [{
      "count": 3,
      "queue": "cartpanda_physical",
      "exchange": "cartpanda_physical.wait.exchange",
      "time": "2025-01-16T10:30:10Z"
    }]
  }
}
```
⚠️ **MAX RETRIES ATINGIDO** → vai para **DLQ**

---

## ⏱️ Timeline de Execução

```
00:00 - Mensagem chega na MAIN QUEUE
00:00 - Consumer processa → FALHA (tentativa 1/3)
00:00 - Vai para WAIT QUEUE
00:05 - TTL expira → vai para RETRY EXCHANGE → volta para MAIN QUEUE
00:05 - Consumer processa → FALHA (tentativa 2/3)
00:05 - Vai para WAIT QUEUE
00:10 - TTL expira → vai para RETRY EXCHANGE → volta para MAIN QUEUE
00:10 - Consumer processa → FALHA (tentativa 3/3)
00:10 - Vai para WAIT QUEUE
00:15 - TTL expira → vai para RETRY EXCHANGE → MAX RETRIES → DLQ ☠️
```

**Tempo total até DLQ: ~15 segundos** (3 tentativas × 5s de espera)

---

## 🔍 Como Verificar no RabbitMQ UI

### 1. Main Queue
```
Nome: cartpanda_physical
Tipo: quorum
Features: D TTL DLX Args
Arguments:
  x-dead-letter-exchange: cartpanda_physical.wait.exchange
```

### 2. Wait Queue
```
Nome: cartpanda_physical.wait
Tipo: classic
Features: D TTL DLX Args
Arguments:
  x-message-ttl: 5000
  x-dead-letter-exchange: cartpanda_physical.retry
```

### 3. Retry Exchange
```
Nome: cartpanda_physical.retry
Tipo: fanout
Features: D
Bindings:
  → cartpanda_physical (se count < 3)
  → cartpanda_physical.dlq (se count >= 3)
```

### 4. DLQ
```
Nome: cartpanda_physical.dlq
Tipo: classic
Features: D TTL Args
Arguments:
  x-message-ttl: 604800000 (7 dias)
```

---

## 🎓 Vantagens desta Arquitetura

✅ **Retry Automático**: Não precisa de código externo para retry
✅ **Configurável**: Fácil alterar número de tentativas e delay
✅ **Observável**: Headers `x-death` mostram histórico de tentativas
✅ **Escalável**: Usa quorum queue para alta disponibilidade
✅ **Seguro**: Mensagens não são perdidas, ficam na DLQ
✅ **Performance**: Wait queue absorve o delay sem bloquear consumers

---

## 🚀 Próximos Passos

1. [ ] Implementar dashboard de monitoramento
2. [ ] Criar alertas para mensagens na DLQ
3. [ ] Implementar re-processing manual da DLQ
4. [ ] Adicionar métricas (Prometheus/Grafana)
5. [ ] Criar testes automatizados de carga
