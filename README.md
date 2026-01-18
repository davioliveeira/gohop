# 🐰 RabbitMQ Dead Letter Queue (DLQ) System

Sistema completo para resolver loops infinitos de execução no n8n causados por falhas no processamento de mensagens do RabbitMQ.

## 📋 O que é Dead Letter Queue (DLQ)?

Uma **Dead Letter Queue** é uma fila especial que armazena mensagens que falharam múltiplas vezes. Isso evita:
- ✅ Loops infinitos de execução
- ✅ Desperdício de recursos computacionais
- ✅ Permite análise posterior de erros
- ✅ Facilita debugging de problemas

## 🚀 Quick Start

### 1. Instalação

```bash
# Instalar dependências
pip install -r requirements.txt

# Configurar credenciais
cp .env.example .env
# Edite o .env se necessário (já vem configurado com suas credenciais)
```

### 2. Ver filas atuais (modo seguro)

```bash
# Ver o que seria feito SEM fazer alterações
python dlq_setup.py --dry-run
```

### 3. Configurar DLQ para todas as filas

```bash
# Isso vai criar as filas mortas
python dlq_setup.py
```

### 4. Monitorar mensagens nas DLQs

```bash
# Ver estatísticas das DLQs
python dlq_monitor.py

# Inspecionar mensagens de uma fila específica
python dlq_monitor.py --inspect nome_da_fila

# Monitoramento contínuo (atualiza a cada 30s)
python dlq_monitor.py --watch
```

### 5. Reprocessar mensagens (quando o problema estiver corrigido)

```bash
# Ver o que seria feito (modo seguro)
python dlq_reprocess.py --queue nome_da_fila --dry-run

# Reprocessar todas as mensagens
python dlq_reprocess.py --queue nome_da_fila

# Reprocessar apenas 10 mensagens (teste)
python dlq_reprocess.py --queue nome_da_fila --max-messages 10
```

## 🔧 Como Funciona

### Antes (Problema)
```
Mensagem → Fila → n8n Workflow → ❌ Falha
     ↑                                  ↓
     └──────────────────────────────────┘
            Loop infinito! 😱
```

### Depois (Solução com DLQ)
```
Mensagem → Fila (tenta 3x) → n8n Workflow
                    ↓
            Ainda falhou?
                    ↓
          Dead Letter Queue
          (fila de mensagens mortas)

Você analisa → Corrige o problema → Reprocessa
```

## 📁 Estrutura do Projeto

```
rabbit/
├── .env                    # Suas credenciais (NÃO commitar!)
├── .env.example           # Template de configuração
├── config.py              # Configurações do RabbitMQ e DLQ
├── dlq_setup.py          # Script para configurar DLQs
├── dlq_monitor.py        # Script para monitorar DLQs
├── dlq_reprocess.py      # Script para reprocessar mensagens
└── requirements.txt       # Dependências Python
```

## 🎯 Processo Completo de Implementação

### Passo 1: Preparação (5 min)
1. Avisar a equipe que você vai fazer manutenção
2. Anotar quais workflows do n8n usam RabbitMQ
3. Ter acesso ao painel do RabbitMQ: http://ec2-52-206-180-123.compute-1.amazonaws.com:15672

### Passo 2: Análise (2 min)
```bash
# Ver todas as filas atuais
python dlq_setup.py --dry-run
```

Isso mostra:
- ✅ Quais filas existem
- ✅ Quantas mensagens cada uma tem
- ✅ Quantos consumidores (n8n workflows) estão conectados

### Passo 3: Criação das DLQs (3 min)
```bash
# Criar as Dead Letter Queues
python dlq_setup.py
```

O script vai criar para cada fila:
- `nome_da_fila.dlq` - Fila morta
- `nome_da_fila.retry` - Exchange de retry

⚠️ **IMPORTANTE**: O script vai avisar que você precisa:
1. Parar os workflows do n8n
2. Deletar a fila original no RabbitMQ
3. Rodar novamente para recriar com DLQ

### Passo 4: Reconfiguração (10 min por fila)

Para cada fila, faça:

```bash
# 1. Parar workflows no n8n que usam a fila
# (fazer manualmente no n8n)

# 2. Deletar a fila no RabbitMQ
# (fazer manualmente no painel web)

# 3. Recriar a fila com DLQ
python dlq_setup.py --recreate nome_da_fila

# 4. Reativar workflows no n8n
# (fazer manualmente no n8n)
```

### Passo 5: Monitoramento (contínuo)

Depois de configurado, monitore regularmente:

```bash
# Verificar se há mensagens nas DLQs
python dlq_monitor.py

# Se houver mensagens, investigar
python dlq_monitor.py --inspect nome_da_fila
```

## 🆘 Cenários de Uso

### Cenário 1: Mensagens estão indo para DLQ
```bash
# 1. Verificar o que está falhando
python dlq_monitor.py --inspect minha_fila

# 2. Corrigir o problema no n8n ou na aplicação

# 3. Testar com algumas mensagens primeiro
python dlq_reprocess.py --queue minha_fila --max-messages 5

# 4. Se funcionou, reprocessar todas
python dlq_reprocess.py --queue minha_fila
```

### Cenário 2: Mensagens inválidas que nunca vão funcionar
```bash
# Deletar permanentemente (cuidado!)
python dlq_reprocess.py --queue minha_fila --purge
```

### Cenário 3: Configurar DLQ para uma fila nova
```bash
# Se a fila já existe
python dlq_setup.py --queue nome_da_fila_nova

# Seguir o processo de reconfiguração
```

## ⚙️ Configurações Avançadas

### Arquivo .env

```bash
# Quantas vezes tentar antes de ir para DLQ
MAX_RETRIES=3

# Tempo de vida das mensagens na fila principal (24h)
MESSAGE_TTL=86400000

# Tempo de vida das mensagens na DLQ (7 dias)
DLQ_MESSAGE_TTL=604800000
```

### Ajustar número de retries

Edite o `.env`:
```bash
MAX_RETRIES=5  # Agora vai tentar 5 vezes antes de ir para DLQ
```

## 🐛 Troubleshooting

### Erro: "Failed to connect to RabbitMQ"
**Solução**: Verificar se as credenciais no `.env` estão corretas

### Erro: "Queue already exists"
**Solução**: A fila já existe. Use `--dry-run` para ver o status atual

### Não vejo mensagens na DLQ mas sei que estão falhando
**Solução**: A DLQ ainda não foi configurada para essa fila. Execute:
```bash
python dlq_setup.py --queue nome_da_fila
```

### Mensagens somem da fila
**Solução**: Provavelmente o TTL expirou. Aumente `MESSAGE_TTL` no `.env`

## 📊 Métricas e Alertas

Configure alertas baseados no script de monitoramento:

```bash
# Exemplo: rodar a cada 5 minutos via cron
*/5 * * * * cd /path/to/rabbit && python dlq_monitor.py | grep "Total messages in DLQs: [1-9]" && echo "ALERTA: Mensagens na DLQ!"
```

## 🔐 Segurança

⚠️ **NUNCA** commite o arquivo `.env` com suas credenciais!

O `.gitignore` já está configurado para ignorar:
- `.env`
- `*.pyc`
- `__pycache__/`

## 💡 Dicas para Apresentar ao Gestor

1. **Mostre o problema atual**:
   ```bash
   # Antes
   "As mensagens ficam em loop infinito, consumindo recursos"
   ```

2. **Mostre a solução**:
   ```bash
   # Ver que agora tem DLQs configuradas
   python dlq_monitor.py
   ```

3. **Mostre que você tem controle**:
   ```bash
   # Pode inspecionar problemas
   python dlq_monitor.py --inspect fila_com_problema

   # Pode reprocessar quando corrigir
   python dlq_reprocess.py --queue fila_com_problema
   ```

4. **Destaque os benefícios**:
   - ✅ Sem mais loops infinitos
   - ✅ Visibilidade total dos erros
   - ✅ Capacidade de reprocessar mensagens
   - ✅ Não perde nenhuma mensagem

## 📚 Conceitos Importantes

### Dead Letter Exchange (DLX)
Quando uma mensagem falha, ela é enviada para um exchange especial (DLX) que roteia para a DLQ.

### TTL (Time To Live)
Tempo que uma mensagem pode ficar na fila antes de expirar.

### Nack/Ack
- **Ack**: Mensagem processada com sucesso
- **Nack**: Mensagem falhou, tentar novamente

### x-death header
Header automático que conta quantas vezes uma mensagem já morreu.

## 🎓 Próximos Passos

1. ✅ Configurar DLQs para todas as filas
2. 📊 Configurar monitoramento automático
3. 🔔 Configurar alertas quando DLQ receber mensagens
4. 📈 Criar dashboard no Grafana (opcional)
5. 📝 Documentar padrões de erro comuns

## 🤝 Suporte

Em caso de dúvidas:
1. Leia este README
2. Use `--dry-run` para testar sem riscos
3. Use `--help` em qualquer script para ver opções

```bash
python dlq_setup.py --help
python dlq_monitor.py --help
python dlq_reprocess.py --help
```

---

**Criado para resolver loops infinitos no n8n + RabbitMQ** 🚀
