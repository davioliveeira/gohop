"""
Monitor Retry Statistics
Shows retry counts and DLQ statistics for queues
"""
import requests
from requests.auth import HTTPBasicAuth
from config import RabbitMQConfig
from tabulate import tabulate
import json


class RetryMonitor:
    """Monitor retry statistics for RabbitMQ queues"""

    def __init__(self):
        self.config = RabbitMQConfig()

    def get_queue_info(self, queue_name):
        """Get detailed queue information"""
        try:
            url = f"{self.config.get_management_url()}/queues/{self.config.VHOST}/{queue_name}"
            response = requests.get(
                url,
                auth=HTTPBasicAuth(self.config.USER, self.config.PASSWORD),
                timeout=10
            )
            response.raise_for_status()
            return response.json()
        except Exception as e:
            return None

    def get_messages(self, queue_name, count=10):
        """Get messages from queue without consuming"""
        try:
            url = f"{self.config.get_management_url()}/queues/{self.config.VHOST}/{queue_name}/get"
            payload = {
                "count": count,
                "ackmode": "ack_requeue_false",
                "encoding": "auto"
            }
            response = requests.post(
                url,
                auth=HTTPBasicAuth(self.config.USER, self.config.PASSWORD),
                json=payload,
                timeout=10
            )
            response.raise_for_status()
            return response.json()
        except Exception as e:
            return []

    def get_retry_count_from_message(self, message):
        """Extract retry count from message properties"""
        properties = message.get('properties', {})
        headers = properties.get('headers', {})
        x_death = headers.get('x-death', [])

        if not x_death:
            return 0

        total_count = sum(death.get('count', 0) for death in x_death)
        return total_count

    def monitor_queue(self, queue_name):
        """Monitor a specific queue and its retry/DLQ"""
        print(f"\n{'='*80}")
        print(f"📊 Monitoring Queue: {queue_name}")
        print(f"{'='*80}\n")

        # Get main queue info
        main_queue = self.get_queue_info(queue_name)
        wait_queue = self.get_queue_info(f"{queue_name}.wait")
        dlq = self.get_queue_info(f"{queue_name}.dlq")

        if not main_queue:
            print(f"❌ Queue '{queue_name}' not found")
            return

        # Display queue statistics
        queue_stats = []

        if main_queue:
            queue_stats.append([
                "Main Queue",
                queue_name,
                main_queue.get('messages', 0),
                main_queue.get('consumers', 0),
                main_queue.get('state', 'unknown')
            ])

        if wait_queue:
            queue_stats.append([
                "Wait Queue",
                f"{queue_name}.wait",
                wait_queue.get('messages', 0),
                wait_queue.get('consumers', 0),
                wait_queue.get('state', 'unknown')
            ])

        if dlq:
            queue_stats.append([
                "DLQ",
                f"{queue_name}.dlq",
                dlq.get('messages', 0),
                dlq.get('consumers', 0),
                dlq.get('state', 'unknown')
            ])

        print(tabulate(
            queue_stats,
            headers=['Type', 'Queue Name', 'Messages', 'Consumers', 'State'],
            tablefmt='grid'
        ))

        # Display DLQ messages with retry counts
        if dlq and dlq.get('messages', 0) > 0:
            print(f"\n{'='*80}")
            print(f"☠️  Messages in DLQ (showing first 10)")
            print(f"{'='*80}\n")

            messages = self.get_messages(f"{queue_name}.dlq", count=10)

            if messages:
                dlq_data = []
                for i, msg in enumerate(messages, 1):
                    retry_count = self.get_retry_count_from_message(msg)
                    payload = msg.get('payload', '')

                    # Try to parse JSON payload
                    try:
                        payload_data = json.loads(payload)
                        payload_preview = str(payload_data)[:50] + "..." if len(str(payload_data)) > 50 else str(payload_data)
                    except:
                        payload_preview = payload[:50] + "..." if len(payload) > 50 else payload

                    dlq_data.append([
                        i,
                        retry_count,
                        payload_preview
                    ])

                print(tabulate(
                    dlq_data,
                    headers=['#', 'Retries', 'Payload Preview'],
                    tablefmt='grid'
                ))
            else:
                print("ℹ️  Could not retrieve messages from DLQ")

        # Display summary
        print(f"\n{'='*80}")
        print(f"📈 Summary")
        print(f"{'='*80}\n")

        total_messages = main_queue.get('messages', 0)
        wait_messages = wait_queue.get('messages', 0) if wait_queue else 0
        dlq_messages = dlq.get('messages', 0) if dlq else 0

        summary = [
            ["Active Messages", total_messages],
            ["Waiting Retry", wait_messages],
            ["Failed (DLQ)", dlq_messages],
            ["Total", total_messages + wait_messages + dlq_messages]
        ]

        print(tabulate(summary, headers=['Metric', 'Count'], tablefmt='grid'))

    def list_all_queues(self):
        """List all queues with retry configuration"""
        try:
            url = f"{self.config.get_management_url()}/queues/{self.config.VHOST}"
            response = requests.get(
                url,
                auth=HTTPBasicAuth(self.config.USER, self.config.PASSWORD),
                timeout=10
            )
            response.raise_for_status()
            queues = response.json()

            # Group queues by base name
            queue_groups = {}
            for q in queues:
                name = q['name']
                if name.endswith('.dlq'):
                    base = name[:-4]
                    if base not in queue_groups:
                        queue_groups[base] = {}
                    queue_groups[base]['dlq'] = q
                elif name.endswith('.wait'):
                    base = name[:-5]
                    if base not in queue_groups:
                        queue_groups[base] = {}
                    queue_groups[base]['wait'] = q
                elif not name.startswith('amq.'):
                    if name not in queue_groups:
                        queue_groups[name] = {}
                    queue_groups[name]['main'] = q

            # Display grouped queues
            print(f"\n{'='*80}")
            print(f"📋 All Queues with Retry Configuration")
            print(f"{'='*80}\n")

            for base_name, queues in queue_groups.items():
                main = queues.get('main')
                wait = queues.get('wait')
                dlq = queues.get('dlq')

                if main:
                    has_dlx = 'x-dead-letter-exchange' in main.get('arguments', {})
                    status = "✅ Configured" if (wait and dlq and has_dlx) else "⚠️  Incomplete"

                    print(f"\n{status} {base_name}")
                    print(f"  Main: {main.get('messages', 0)} msgs | {main.get('consumers', 0)} consumers")
                    if wait:
                        print(f"  Wait: {wait.get('messages', 0)} msgs (retrying)")
                    if dlq:
                        print(f"  DLQ:  {dlq.get('messages', 0)} msgs (failed)")

        except Exception as e:
            print(f"❌ Failed to list queues: {e}")


def main():
    """Main execution function"""
    import argparse

    parser = argparse.ArgumentParser(
        description='Monitor retry statistics for RabbitMQ queues'
    )
    parser.add_argument(
        '--queue',
        type=str,
        help='Monitor specific queue'
    )
    parser.add_argument(
        '--list',
        action='store_true',
        help='List all queues with retry configuration'
    )

    args = parser.parse_args()

    monitor = RetryMonitor()

    if args.list:
        monitor.list_all_queues()
    elif args.queue:
        monitor.monitor_queue(args.queue)
    else:
        print("❌ Please specify --queue or --list")


if __name__ == '__main__':
    main()
