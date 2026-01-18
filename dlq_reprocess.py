"""
Dead Letter Queue Reprocessing Script
Move messages from DLQ back to original queue for reprocessing
"""
import pika
import sys
from config import RabbitMQConfig, DLQConfig


class DLQReprocessor:
    """Reprocess messages from Dead Letter Queues"""

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

    def reprocess_messages(self, queue_name, max_messages=None, dry_run=False):
        """
        Move messages from DLQ back to original queue

        Args:
            queue_name: Original queue name
            max_messages: Maximum number of messages to reprocess (None = all)
            dry_run: If True, only show what would be done
        """
        dlq_name = self.dlq_config.get_dlq_name(queue_name)

        print(f"\n{'='*80}")
        print(f"{'[DRY RUN] ' if dry_run else ''}Reprocessing messages from DLQ")
        print(f"  DLQ: {dlq_name}")
        print(f"  Target Queue: {queue_name}")
        print(f"  Max Messages: {max_messages or 'All'}")
        print(f"{'='*80}\n")

        processed_count = 0
        failed_count = 0

        while True:
            # Check if we've reached the limit
            if max_messages and processed_count >= max_messages:
                break

            # Get message from DLQ
            method_frame, properties, body = self.channel.basic_get(
                queue=dlq_name,
                auto_ack=False
            )

            if method_frame is None:
                # No more messages
                break

            try:
                if not dry_run:
                    # Publish message back to original queue
                    # Remove death headers to reset retry count
                    new_headers = properties.headers.copy() if properties.headers else {}
                    new_headers.pop('x-death', None)

                    new_properties = pika.BasicProperties(
                        delivery_mode=2,  # Persistent
                        headers=new_headers,
                        content_type=properties.content_type,
                        content_encoding=properties.content_encoding,
                        correlation_id=properties.correlation_id,
                        reply_to=properties.reply_to,
                        expiration=properties.expiration,
                        message_id=properties.message_id,
                        timestamp=properties.timestamp,
                        type=properties.type,
                        user_id=properties.user_id,
                        app_id=properties.app_id
                    )

                    self.channel.basic_publish(
                        exchange='',
                        routing_key=queue_name,
                        body=body,
                        properties=new_properties
                    )

                    # Acknowledge the message from DLQ
                    self.channel.basic_ack(delivery_tag=method_frame.delivery_tag)

                    print(f"✅ Reprocessed message {processed_count + 1}")
                else:
                    print(f"🔍 Would reprocess message {processed_count + 1}")
                    # In dry run, nack and requeue the message
                    self.channel.basic_nack(
                        delivery_tag=method_frame.delivery_tag,
                        requeue=True
                    )

                processed_count += 1

            except Exception as e:
                print(f"❌ Failed to reprocess message: {e}")
                # Nack and requeue the message
                self.channel.basic_nack(
                    delivery_tag=method_frame.delivery_tag,
                    requeue=True
                )
                failed_count += 1

        print(f"\n{'='*80}")
        print(f"Reprocessing Summary:")
        print(f"  ✅ Successfully reprocessed: {processed_count}")
        if failed_count > 0:
            print(f"  ❌ Failed: {failed_count}")
        print(f"{'='*80}\n")

        if not dry_run and processed_count > 0:
            print("⚠️  Messages have been moved back to the original queue.")
            print("   Monitor n8n workflows to ensure they process correctly.")

    def purge_dlq(self, queue_name, confirm=True):
        """
        Permanently delete all messages from a DLQ
        WARNING: This action cannot be undone!
        """
        dlq_name = self.dlq_config.get_dlq_name(queue_name)

        if confirm:
            print(f"\n⚠️  WARNING: You are about to permanently delete ALL messages from:")
            print(f"   {dlq_name}")
            print(f"\n   This action CANNOT be undone!")
            response = input("\n   Type 'DELETE' to confirm: ")

            if response != 'DELETE':
                print("\n   Purge cancelled.")
                return

        try:
            result = self.channel.queue_purge(queue=dlq_name)
            print(f"\n✅ Purged {result.method.message_count} messages from {dlq_name}")
        except Exception as e:
            print(f"\n❌ Failed to purge DLQ: {e}")

    def close(self):
        """Close RabbitMQ connection"""
        if self.connection and not self.connection.is_closed:
            self.connection.close()


def main():
    """Main execution function"""
    import argparse

    parser = argparse.ArgumentParser(
        description='Reprocess messages from Dead Letter Queues'
    )
    parser.add_argument(
        '--queue',
        type=str,
        required=True,
        help='Original queue name (not DLQ name)'
    )
    parser.add_argument(
        '--max-messages',
        type=int,
        help='Maximum number of messages to reprocess'
    )
    parser.add_argument(
        '--dry-run',
        action='store_true',
        help='Show what would be done without making changes'
    )
    parser.add_argument(
        '--purge',
        action='store_true',
        help='Permanently delete all messages from DLQ (dangerous!)'
    )

    args = parser.parse_args()

    reprocessor = DLQReprocessor()

    if not reprocessor.connect():
        sys.exit(1)

    try:
        if args.purge:
            reprocessor.purge_dlq(args.queue)
        else:
            reprocessor.reprocess_messages(
                args.queue,
                args.max_messages,
                args.dry_run
            )
    finally:
        reprocessor.close()


if __name__ == '__main__':
    main()
