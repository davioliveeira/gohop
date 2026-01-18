"""
Setup DLQ for priority queues
Configure Dead Letter Queues for specific business-critical queues
"""
import sys
from dlq_setup import DLQSetup
from tabulate import tabulate


# Priority queue keywords
PRIORITY_KEYWORDS = [
    'active',
    'cartpanda',
    'buygoods',
    'digistore',
    'google',
    'meta',
    'slicktext',
    'salesbound',
    'logicall',
    'chatwoot'
]


def get_priority_queues(setup):
    """Get queues matching priority keywords"""
    all_queues = setup.get_all_queues()

    priority_queues = []
    for queue in all_queues:
        queue_name_lower = queue['name'].lower()
        if any(keyword in queue_name_lower for keyword in PRIORITY_KEYWORDS):
            priority_queues.append(queue)

    return priority_queues


def main():
    """Main execution"""
    import argparse

    parser = argparse.ArgumentParser(
        description='Setup DLQ for priority queues (active, cartpanda, buygoods, etc.)'
    )
    parser.add_argument(
        '--dry-run',
        action='store_true',
        help='Show what would be done without making changes'
    )
    parser.add_argument(
        '--show-only',
        action='store_true',
        help='Only show the list of queues, do not setup'
    )
    parser.add_argument(
        '--top',
        type=int,
        default=None,
        help='Setup DLQ only for top N queues with most messages'
    )

    args = parser.parse_args()

    setup = DLQSetup()

    if not setup.connect():
        sys.exit(1)

    try:
        priority_queues = get_priority_queues(setup)

        if not priority_queues:
            print("No priority queues found.")
            return

        # Sort by message count (descending)
        priority_queues.sort(key=lambda q: q.get('messages', 0), reverse=True)

        # Filter top N if specified
        if args.top:
            priority_queues = priority_queues[:args.top]

        print(f"\n{'='*80}")
        print(f"Found {len(priority_queues)} priority queue(s)")
        print(f"Keywords: {', '.join(PRIORITY_KEYWORDS)}")
        print(f"{'='*80}\n")

        # Display table
        table_data = []
        total_messages = 0
        for q in priority_queues:
            msgs = q.get('messages', 0)
            total_messages += msgs
            table_data.append([
                q['name'],
                msgs,
                q.get('consumers', 0),
                '🔥' if msgs > 10000 else ('⚠️' if msgs > 1000 else ('✓' if msgs > 0 else ''))
            ])

        print(tabulate(
            table_data,
            headers=['Queue Name', 'Messages', 'Consumers', 'Status'],
            tablefmt='grid'
        ))

        print(f"\n📊 Total messages in priority queues: {total_messages:,}")

        if args.show_only:
            print("\n💡 To setup DLQ for these queues, run without --show-only flag")
            return

        if args.dry_run:
            print("\n🔍 DRY RUN MODE - No changes will be made\n")
        else:
            print("\n⚠️  WARNING: This will create DLQ infrastructure for these queues.")
            print("   Note: To complete the setup, you'll need to:")
            print("   1. Stop n8n workflows using these queues")
            print("   2. Delete the queues in RabbitMQ UI")
            print("   3. Run: python dlq_setup.py --recreate <queue_name>")
            print("   4. Restart n8n workflows\n")

            confirm = input("Continue? (yes/no): ")
            if confirm.lower() != 'yes':
                print("Setup cancelled.")
                return

        # Setup DLQ for each queue
        success_count = 0
        for queue in priority_queues:
            if setup.setup_dlq_for_queue(queue['name'], args.dry_run):
                success_count += 1

        print(f"\n{'='*80}")
        print(f"✅ Successfully configured {success_count}/{len(priority_queues)} queues")
        print(f"{'='*80}")

    finally:
        setup.close()


if __name__ == '__main__':
    main()
