"""
Configuration module for RabbitMQ DLQ system
"""
import os
from dotenv import load_dotenv

load_dotenv()


class RabbitMQConfig:
    """RabbitMQ connection configuration"""

    HOST = os.getenv('RABBITMQ_HOST', 'localhost')
    PORT = int(os.getenv('RABBITMQ_PORT', 5672))
    MANAGEMENT_PORT = int(os.getenv('RABBITMQ_MANAGEMENT_PORT', 15672))
    USER = os.getenv('RABBITMQ_USER', 'guest')
    PASSWORD = os.getenv('RABBITMQ_PASSWORD', 'guest')
    VHOST = os.getenv('RABBITMQ_VHOST', '/')

    @classmethod
    def get_connection_params(cls):
        """Returns pika connection parameters"""
        import pika
        credentials = pika.PlainCredentials(cls.USER, cls.PASSWORD)
        return pika.ConnectionParameters(
            host=cls.HOST,
            port=cls.PORT,
            virtual_host=cls.VHOST,
            credentials=credentials,
            heartbeat=600,
            blocked_connection_timeout=300
        )

    @classmethod
    def get_management_url(cls):
        """Returns RabbitMQ management API base URL"""
        return f"http://{cls.HOST}:{cls.MANAGEMENT_PORT}/api"


class DLQConfig:
    """Dead Letter Queue configuration"""

    MAX_RETRIES = int(os.getenv('MAX_RETRIES', 3))
    MESSAGE_TTL = int(os.getenv('MESSAGE_TTL', 86400000))  # 24 hours
    DLQ_MESSAGE_TTL = int(os.getenv('DLQ_MESSAGE_TTL', 604800000))  # 7 days

    @staticmethod
    def get_dlq_name(queue_name):
        """Generate DLQ name from original queue name"""
        return f"{queue_name}.dlq"

    @staticmethod
    def get_retry_exchange_name(queue_name):
        """Generate retry exchange name from original queue name"""
        return f"{queue_name}.retry"
