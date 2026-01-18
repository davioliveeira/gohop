"""
Example Consumer with Retry Logic
This shows how to implement retry counting with RabbitMQ
"""
import pika
import json
from config import RabbitMQConfig, DLQConfig


class ConsumerWithRetry:
    """Consumer that handles retry logic with max attempts"""

    def __init__(self, queue_name):
        self.config = RabbitMQConfig()
        self.dlq_config = DLQConfig()
        self.queue_name = queue_name
        self.connection = None
        self.channel = None

    def connect(self):
        """Establish connection to RabbitMQ"""
        self.connection = pika.BlockingConnection(
            self.config.get_connection_params()
        )
        self.channel = self.connection.channel()
        print(f"✅ Connected to queue: {self.queue_name}")

    def get_retry_count(self, properties):
        """
        Extract retry count from message headers
        RabbitMQ tracks this in x-death header automatically
        """
        if not properties.headers:
            return 0

        # Check x-death header (automatically added by RabbitMQ)
        x_death = properties.headers.get('x-death', [])
        if not x_death:
            return 0

        # Sum up all the counts from x-death
        total_count = sum(death.get('count', 0) for death in x_death)
        return total_count

    def should_retry(self, properties):
        """Check if message should be retried or sent to DLQ"""
        retry_count = self.get_retry_count(properties)
        return retry_count < self.dlq_config.MAX_RETRIES

    def process_message(self, ch, method, properties, body):
        """
        Process message with retry logic

        Args:
            ch: Channel
            method: Delivery method
            properties: Message properties
            body: Message body
        """
        retry_count = self.get_retry_count(properties)

        try:
            # Your business logic here
            data = json.loads(body)
            print(f"\n📨 Processing message (attempt {retry_count + 1}/{self.dlq_config.MAX_RETRIES})")
            print(f"   Data: {data}")

            # Simulate processing that might fail
            # Replace this with your actual logic
            if data.get('fail', False):
                raise Exception("Simulated processing error")

            # Success - acknowledge the message
            ch.basic_ack(delivery_tag=method.delivery_tag)
            print("   ✅ Message processed successfully")

        except Exception as e:
            print(f"   ❌ Error processing message: {e}")

            # Check if we should retry
            if self.should_retry(properties):
                # Reject and requeue (will go to wait queue -> retry exchange -> back to main queue)
                print(f"   🔄 Retrying... (attempt {retry_count + 1}/{self.dlq_config.MAX_RETRIES})")
                ch.basic_reject(delivery_tag=method.delivery_tag, requeue=False)
            else:
                # Max retries reached - send to DLQ
                print(f"   ⚠️  Max retries reached. Sending to DLQ.")
                ch.basic_reject(delivery_tag=method.delivery_tag, requeue=False)

    def start_consuming(self):
        """Start consuming messages"""
        self.channel.basic_qos(prefetch_count=1)
        self.channel.basic_consume(
            queue=self.queue_name,
            on_message_callback=self.process_message
        )

        print(f"\n👂 Waiting for messages on queue: {self.queue_name}")
        print(f"   Max retries: {self.dlq_config.MAX_RETRIES}")
        print("   Press CTRL+C to exit\n")

        try:
            self.channel.start_consuming()
        except KeyboardInterrupt:
            print("\n\n👋 Stopping consumer...")
            self.channel.stop_consuming()
        finally:
            if self.connection:
                self.connection.close()


def main():
    """Main execution function"""
    import argparse

    parser = argparse.ArgumentParser(
        description='Consumer with retry logic for RabbitMQ'
    )
    parser.add_argument(
        '--queue',
        type=str,
        required=True,
        help='Queue name to consume from'
    )

    args = parser.parse_args()

    consumer = ConsumerWithRetry(args.queue)
    consumer.connect()
    consumer.start_consuming()


if __name__ == '__main__':
    main()
