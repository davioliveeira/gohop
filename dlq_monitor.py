"""
Dead Letter Queue Monitoring Script
Monitor and manage messages in DLQs
"""
import pika
import json
import sys
from datetime import datetime
from config import RabbitMQConfig, DLQConfig
from tabulate import tabulate


class DLQMonitor:
    """Monitor Dead Letter Queues"""

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
            return True
        except Exception as e:
            print(f"❌ Failed to connect to RabbitMQ: {e}")
            return False

    def get_dlq_stats(self):
        """Get statistics for all DLQs"""
        import requests
        from requests.auth import HTTPBasicAuth

        try:
            url = f"{self.config.get_management_url()}/queues/{self.config.VHOST}"
            response = requests.get(
                url,
                auth=HTTPBasicAuth(self.config.USER, self.config.PASSWORD),
                timeout=10
            )
            response.raise_for_status()
            queues = response.json()

            # Filter only DLQs
            dlqs = [q for q in queues if q['name'].endswith('.dlq')]

            return dlqs
        except Exception as e:
            print(f"❌ Failed to get DLQ stats: {e}")
            return []

    def display_dlq_stats(self):
        """Display DLQ statistics in table format"""
        dlqs = self.get_dlq_stats()

        if not dlqs:
            print("No Dead Letter Queues found or no messages in DLQs.")
            return

        print(f"\n{'='*80}")
        print(f"Dead Letter Queue Statistics - {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
        print(f"{'='*80}\n")

        table_data = []
        total_messages = 0

        for dlq in dlqs:
            messages = dlq.get('messages', 0)
            total_messages += messages

            original_queue = dlq['name'].replace('.dlq', '')

            table_data.append([
                original_queue,
                dlq['name'],
                messages,
                dlq.get('messages_ready', 0),
                dlq.get('messages_unacknowledged', 0),
                dlq.get('consumers', 0)
            ])

        print(tabulate(
            table_data,
            headers=[
                'Original Queue',
                'DLQ Name',
                'Total Messages',
                'Ready',
                'Unacked',
                'Consumers'
            ],
            tablefmt='grid'
        ))

        print(f"\n📊 Total messages in DLQs: {total_messages}")

        if total_messages > 0:
            print("\n⚠️  Action Required: Messages in DLQs need attention!")
            print("   Use 'python dlq_monitor.py --inspect <queue_name>' to view messages")
            print("   Use 'python dlq_reprocess.py --queue <queue_name>' to reprocess")

    def inspect_dlq(self, queue_name):
        """Inspect messages in a specific DLQ"""
        dlq_name = self.dlq_config.get_dlq_name(queue_name)

        print(f"\n{'='*80}")
        print(f"Inspecting DLQ: {dlq_name}")
        print(f"{'='*80}\n")

        messages = []
        message_count = 0

        # Get messages without consuming them
        while True:
            method_frame, properties, body = self.channel.basic_get(
                queue=dlq_name,
                auto_ack=False
            )

            if method_frame is None:
                break

            message_count += 1

            try:
                body_decoded = body.decode('utf-8')
                try:
                    body_json = json.loads(body_decoded)
                    body_preview = json.dumps(body_json, indent=2)[:200]
                except:
                    body_preview = body_decoded[:200]
            except:
                body_preview = str(body)[:200]

            messages.append({
                'number': message_count,
                'delivery_tag': method_frame.delivery_tag,
                'exchange': method_frame.exchange,
                'routing_key': method_frame.routing_key,
                'headers': properties.headers or {},
                'body_preview': body_preview
            })

            # Requeue the message (don't consume it)
            self.channel.basic_nack(
                delivery_tag=method_frame.delivery_tag,
                requeue=True
            )

            # Limit inspection to first 10 messages
            if message_count >= 10:
                break

        if not messages:
            print("No messages found in DLQ.")
            return

        for msg in messages:
            print(f"Message #{msg['number']}")
            print(f"  Delivery Tag: {msg['delivery_tag']}")
            print(f"  Exchange: {msg['exchange']}")
            print(f"  Routing Key: {msg['routing_key']}")
            print(f"  Headers: {json.dumps(msg['headers'], indent=4)}")
            print(f"  Body Preview: {msg['body_preview']}")
            print(f"  {'─'*76}\n")

        if message_count >= 10:
            print(f"⚠️  Showing first 10 messages only. Total messages may be more.")

    def close(self):
        """Close RabbitMQ connection"""
        if self.connection and not self.connection.is_closed:
            self.connection.close()


def main():
    """Main execution function"""
    import argparse

    parser = argparse.ArgumentParser(
        description='Monitor Dead Letter Queues'
    )
    parser.add_argument(
        '--inspect',
        type=str,
        help='Inspect messages in DLQ for specific queue'
    )
    parser.add_argument(
        '--watch',
        action='store_true',
        help='Continuously watch DLQ statistics (updates every 30s)'
    )

    args = parser.parse_args()

    monitor = DLQMonitor()

    if not monitor.connect():
        sys.exit(1)

    try:
        if args.inspect:
            monitor.inspect_dlq(args.inspect)
        elif args.watch:
            import time
            print("👀 Watching DLQ statistics... (Press Ctrl+C to stop)")
            while True:
                monitor.display_dlq_stats()
                time.sleep(30)
        else:
            monitor.display_dlq_stats()
    except KeyboardInterrupt:
        print("\n\n👋 Monitoring stopped.")
    finally:
        monitor.close()


if __name__ == '__main__':
    main()
