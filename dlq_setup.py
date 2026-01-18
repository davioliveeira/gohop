"""
Dead Letter Queue Setup Script
This script configures DLQ for existing RabbitMQ queues
"""
import pika
import sys
import requests
from requests.auth import HTTPBasicAuth
from config import RabbitMQConfig, DLQConfig
from tabulate import tabulate


class DLQSetup:
    """Setup Dead Letter Queues for RabbitMQ"""

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

    def get_all_queues(self):
        """Get all queues from RabbitMQ Management API"""
        try:
            url = f"{self.config.get_management_url()}/queues/{self.config.VHOST}"
            response = requests.get(
                url,
                auth=HTTPBasicAuth(self.config.USER, self.config.PASSWORD),
                timeout=10
            )
            response.raise_for_status()
            queues = response.json()

            # Filter out DLQ and system queues
            return [
                q for q in queues
                if not q['name'].endswith('.dlq')
                and not q['name'].startswith('amq.')
            ]
        except Exception as e:
            print(f"❌ Failed to get queues: {e}")
            return []

    def setup_dlq_for_queue(self, queue_name, dry_run=False):
        """
        Setup DLQ configuration for a specific queue

        Args:
            queue_name: Name of the queue to configure
            dry_run: If True, only show what would be done
        """
        dlq_name = self.dlq_config.get_dlq_name(queue_name)
        retry_exchange = self.dlq_config.get_retry_exchange_name(queue_name)

        print(f"\n{'[DRY RUN] ' if dry_run else ''}Configuring DLQ for queue: {queue_name}")
        print(f"  DLQ Name: {dlq_name}")
        print(f"  Retry Exchange: {retry_exchange}")

        if dry_run:
            return True

        try:
            # 1. Create DLQ (Dead Letter Queue)
            self.channel.queue_declare(
                queue=dlq_name,
                durable=True,
                arguments={
                    'x-message-ttl': self.dlq_config.DLQ_MESSAGE_TTL,
                    'x-queue-type': 'quorum'
                }
            )
            print(f"  ✅ Created DLQ: {dlq_name}")

            # 2. Create retry exchange
            self.channel.exchange_declare(
                exchange=retry_exchange,
                exchange_type='fanout',
                durable=True
            )
            print(f"  ✅ Created retry exchange: {retry_exchange}")

            # 3. Bind DLQ to retry exchange
            self.channel.queue_bind(
                queue=dlq_name,
                exchange=retry_exchange
            )
            print(f"  ✅ Bound DLQ to retry exchange")

            # 4. Get current queue configuration
            print(f"  ⚠️  NOTE: To complete setup, you need to:")
            print(f"     1. Stop all n8n workflows using queue '{queue_name}'")
            print(f"     2. Delete the queue '{queue_name}' from RabbitMQ UI")
            print(f"     3. Run this script again with --apply flag")
            print(f"     4. Restart n8n workflows")

            return True

        except Exception as e:
            print(f"  ❌ Failed to setup DLQ: {e}")
            return False

    def recreate_queue_with_dlq(self, queue_name):
        """
        Recreate a queue with DLQ configuration
        WARNING: Only use this if the queue has been deleted manually first
        """
        dlq_name = self.dlq_config.get_dlq_name(queue_name)
        retry_exchange = self.dlq_config.get_retry_exchange_name(queue_name)

        try:
            # Create main queue with DLQ configuration
            self.channel.queue_declare(
                queue=queue_name,
                durable=True,
                arguments={
                    'x-dead-letter-exchange': retry_exchange,
                    'x-message-ttl': self.dlq_config.MESSAGE_TTL,
                    'x-queue-type': 'quorum'
                }
            )
            print(f"  ✅ Recreated queue '{queue_name}' with DLQ configuration")
            return True
        except Exception as e:
            print(f"  ❌ Failed to recreate queue: {e}")
            return False

    def setup_all_queues(self, dry_run=False):
        """Setup DLQ for all queues"""
        queues = self.get_all_queues()

        if not queues:
            print("No queues found or failed to retrieve queues.")
            return

        print(f"\n{'='*60}")
        print(f"Found {len(queues)} queue(s) to configure:")
        print(f"{'='*60}")

        # Display queues in table format
        table_data = []
        for q in queues:
            table_data.append([
                q['name'],
                q.get('messages', 0),
                q.get('consumers', 0)
            ])

        print(tabulate(
            table_data,
            headers=['Queue Name', 'Messages', 'Consumers'],
            tablefmt='grid'
        ))

        if dry_run:
            print("\n🔍 DRY RUN MODE - No changes will be made")
        else:
            confirm = input("\n⚠️  Continue with DLQ setup? (yes/no): ")
            if confirm.lower() != 'yes':
                print("Setup cancelled.")
                return

        # Setup DLQ for each queue
        success_count = 0
        for queue in queues:
            if self.setup_dlq_for_queue(queue['name'], dry_run):
                success_count += 1

        print(f"\n{'='*60}")
        print(f"✅ Successfully configured {success_count}/{len(queues)} queues")
        print(f"{'='*60}")

    def close(self):
        """Close RabbitMQ connection"""
        if self.connection and not self.connection.is_closed:
            self.connection.close()
            print("\n👋 Connection closed.")


def main():
    """Main execution function"""
    import argparse

    parser = argparse.ArgumentParser(
        description='Setup Dead Letter Queues for RabbitMQ'
    )
    parser.add_argument(
        '--dry-run',
        action='store_true',
        help='Show what would be done without making changes'
    )
    parser.add_argument(
        '--queue',
        type=str,
        help='Setup DLQ for specific queue only'
    )
    parser.add_argument(
        '--recreate',
        type=str,
        help='Recreate a specific queue with DLQ (queue must be deleted first)'
    )

    args = parser.parse_args()

    setup = DLQSetup()

    if not setup.connect():
        sys.exit(1)

    try:
        if args.recreate:
            setup.recreate_queue_with_dlq(args.recreate)
        elif args.queue:
            setup.setup_dlq_for_queue(args.queue, args.dry_run)
        else:
            setup.setup_all_queues(args.dry_run)
    finally:
        setup.close()


if __name__ == '__main__':
    main()
