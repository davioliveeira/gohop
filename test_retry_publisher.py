"""
Test Publisher for Retry Logic
Publishes test messages to verify retry mechanism
"""
import pika
import json
from config import RabbitMQConfig


class TestPublisher:
    """Publisher for testing retry logic"""

    def __init__(self, queue_name):
        self.config = RabbitMQConfig()
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

    def publish_success_message(self):
        """Publish a message that will succeed"""
        message = {
            'id': 'test-success-001',
            'data': 'This message will succeed',
            'fail': False
        }

        self.channel.basic_publish(
            exchange='',
            routing_key=self.queue_name,
            body=json.dumps(message),
            properties=pika.BasicProperties(
                delivery_mode=2,  # Persistent
                content_type='application/json'
            )
        )
        print(f"✅ Published SUCCESS message: {message['id']}")

    def publish_fail_message(self):
        """Publish a message that will fail and trigger retries"""
        message = {
            'id': 'test-fail-001',
            'data': 'This message will fail and retry',
            'fail': True
        }

        self.channel.basic_publish(
            exchange='',
            routing_key=self.queue_name,
            body=json.dumps(message),
            properties=pika.BasicProperties(
                delivery_mode=2,  # Persistent
                content_type='application/json'
            )
        )
        print(f"📤 Published FAIL message: {message['id']} (will retry 3x)")

    def close(self):
        """Close connection"""
        if self.connection:
            self.connection.close()
            print("\n👋 Connection closed.")


def main():
    """Main execution function"""
    import argparse

    parser = argparse.ArgumentParser(
        description='Test publisher for retry logic'
    )
    parser.add_argument(
        '--queue',
        type=str,
        required=True,
        help='Queue name to publish to'
    )
    parser.add_argument(
        '--type',
        type=str,
        choices=['success', 'fail'],
        default='fail',
        help='Type of message to publish (success or fail)'
    )

    args = parser.parse_args()

    publisher = TestPublisher(args.queue)
    publisher.connect()

    try:
        if args.type == 'success':
            publisher.publish_success_message()
        else:
            publisher.publish_fail_message()
    finally:
        publisher.close()


if __name__ == '__main__':
    main()
