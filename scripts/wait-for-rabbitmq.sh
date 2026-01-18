#!/bin/bash
# Script para aguardar RabbitMQ estar pronto

set -e

HOST=${RABBITMQ_HOST:-localhost}
PORT=${RABBITMQ_PORT:-5672}
MAX_ATTEMPTS=${MAX_ATTEMPTS:-30}
ATTEMPT=0

echo "⏳ Aguardando RabbitMQ em ${HOST}:${PORT}..."

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    if nc -z "$HOST" "$PORT" 2>/dev/null; then
        echo "✅ RabbitMQ está pronto!"
        exit 0
    fi
    
    ATTEMPT=$((ATTEMPT + 1))
    echo "   Tentativa $ATTEMPT/$MAX_ATTEMPTS..."
    sleep 2
done

echo "❌ Timeout aguardando RabbitMQ"
exit 1
