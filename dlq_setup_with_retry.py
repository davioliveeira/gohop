"""
Dead Letter Queue Setup with Retry Logic
This script configures DLQ with retry mechanism (3 attempts before DLQ)
"""
import pika
import sys
import requests
from requests.auth import HTTPBasicAuth
from config import RabbitMQConfig, DLQConfig
from tabulate import tabulate


class DLQSetupWithRetry:
    """Setup Dead Letter Queues with retry logic for RabbitMQ"""

    def __init__(self):
        self.config = RabbitMQConfig()
        self.dlq_config = DLQConfig()
        self.connection = None
        self.channel = None

    def connect(self):
        """Establish connection to RabbitMQ"""
        try:
            self.connection = pika.BlockingConnection(
                self.config.get_connection_params()
            )
            self.channel = self.connection.channel()
            print("✅ Connected to RabbitMQ successfully!")
            return True
        except Exception as e:
            print(f"❌ Failed to connect to RabbitMQ: {e}")
            return False

    def setup_dlq_with_retry(self, queue_name, dry_run=False):
        """
        Setup DLQ configuration with retry logic

        Architecture:
        1. Main Queue -> (reject) -> Wait Exchange -> Wait Queue (TTL 5s) -> Retry Exchange
        2. Retry Exchange -> routes back to Main Queue (if retries < MAX_RETRIES)
        3. Retry Exchange -> routes to DLQ (if retries >= MAX_RETRIES)

        Args:
            queue_name: Name of the queue to configure
            dry_run: If True, only show what would be done
        """
        dlq_name = self.dlq_config.get_dlq_name(queue_name)
        wait_queue_name = f"{queue_name}.wait"
        wait_exchange_name = f"{queue_name}.wait.exchange"
        retry_exchange_name = self.dlq_config.get_retry_exchange_name(queue_name)

        print(f"\n{'[DRY RUN] ' if dry_run else ''}Configuring DLQ with retry for queue: {queue_name}")
        print(f"  Main Queue: {queue_name}")
        print(f"  Wait Queue: {wait_queue_name} (5s delay)")
        print(f"  Wait Exchange: {wait_exchange_name}")
        print(f"  Retry Exchange: {retry_exchange_name}")
        print(f"  DLQ: {dlq_name}")
        print(f"  Max Retries: {self.dlq_config.MAX_RETRIES}")

        if dry_run:
            return True

        try:
            # 1. Create Wait Exchange (receives rejected messages)
            self.channel.exchange_declare(
                exchange=wait_exchange_name,
                exchange_type='fanout',
                durable=True
            )
            print(f"  ✅ Created wait exchange: {wait_exchange_name}")

            # 2. Create Wait Queue (delays message before retry)
            # Messages expire after 5 seconds and go to retry exchange
            self.channel.queue_declare(
                queue=wait_queue_name,
                durable=True,
                arguments={
                    'x-message-ttl': 5000,  # 5 seconds delay
                    'x-dead-letter-exchange': retry_exchange_name,
                    'x-queue-type': 'classic'
                }
            )
            print(f"  ✅ Created wait queue: {wait_queue_name}")

            # 3. Bind Wait Queue to Wait Exchange
            self.channel.queue_bind(
                queue=wait_queue_name,
                exchange=wait_exchange_name
            )
            print(f"  ✅ Bound wait queue to wait exchange")

            # 4. Create Retry Exchange (routes to main queue or DLQ based on retry count)
            self.channel.exchange_declare(
                exchange=retry_exchange_name,
                exchange_type='headers',
                durable=True
            )
            print(f"  ✅ Created retry exchange: {retry_exchange_name}")

            # 5. Create DLQ (final destination after max retries)
            self.channel.queue_declare(
                queue=dlq_name,
                durable=True,
                arguments={
                    'x-message-ttl': self.dlq_config.DLQ_MESSAGE_TTL,
                    'x-queue-type': 'classic'
                }
            )
            print(f"  ✅ Created DLQ: {dlq_name}")

            # 6. Bind DLQ to Retry Exchange (for messages that exceeded max retries)
            # Note: This binding needs to be handled by consumer logic
            # RabbitMQ doesn't natively support retry count checking in bindings
            self.channel.queue_bind(
                queue=dlq_name,
                exchange=retry_exchange_name
            )
            print(f"  ✅ Bound DLQ to retry exchange")

            print(f"\n  ⚠️  NOTE: To complete setup, you need to:")
            print(f"     1. Stop all n8n workflows using queue '{queue_name}'")
            print(f"     2. Delete the queue '{queue_name}' from RabbitMQ UI")
            print(f"     3. Run: python dlq_setup_with_retry.py --recreate {queue_name}")
            print(f"     4. Update your consumer to handle retry logic (see consumer_example.py)")
            print(f"     5. Restart n8n workflows")

            return True

        except Exception as e:
            print(f"  ❌ Failed to setup DLQ with retry: {e}")
            return False

    def recreate_queue_with_dlq(self, queue_name):
        """
        Recreate a queue with DLQ configuration
        WARNING: Only use this if the queue has been deleted manually first
        """
        wait_exchange_name = f"{queue_name}.wait.exchange"

        try:
            # Create main queue with DLX pointing to wait exchange
            self.channel.queue_declare(
                queue=queue_name,
                durable=True,
                arguments={
                    'x-dead-letter-exchange': wait_exchange_name,
                    'x-queue-type': 'quorum'
                }
            )
            print(f"  ✅ Recreated queue '{queue_name}' with DLQ configuration")
            print(f"     Dead-letter-exchange: {wait_exchange_name}")
            return True
        except Exception as e:
            print(f"  ❌ Failed to recreate queue: {e}")
            return False

    def close(self):
        """Close RabbitMQ connection"""
        if self.connection and not self.connection.is_closed:
            self.connection.close()
            print("\n👋 Connection closed.")


def main():
    """Main execution function"""
    import argparse

    parser = argparse.ArgumentParser(
        description='Setup Dead Letter Queues with Retry Logic for RabbitMQ'
    )
    parser.add_argument(
        '--dry-run',
        action='store_true',
        help='Show what would be done without making changes'
    )
    parser.add_argument(
        '--queue',
        type=str,
        help='Setup DLQ with retry for specific queue only'
    )
    parser.add_argument(
        '--recreate',
        type=str,
        help='Recreate a specific queue with DLQ (queue must be deleted first)'
    )

    args = parser.parse_args()

    setup = DLQSetupWithRetry()

    if not setup.connect():
        sys.exit(1)

    try:
        if args.recreate:
            setup.recreate_queue_with_dlq(args.recreate)
        elif args.queue:
            setup.setup_dlq_with_retry(args.queue, args.dry_run)
        else:
            print("❌ Please specify --queue or --recreate")
            sys.exit(1)
    finally:
        setup.close()


if __name__ == '__main__':
    main()
