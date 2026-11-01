"""TEST ONLY D-104 platform qualification, separate from the packet predicate."""


def check_precision(values):
    # These stored durations round-trip through the kernel's milliseconds ->
    # jiffies -> milliseconds conversion, independent of allocation/read time.
    # All eight exact values are the 1 ms quantum proof (JSON loses this data).
    assert len(values) == 8 and [a for a, _ in values] == list(range(1001, 1009)), values
    # Each explicit expiration is a countdown from its OWN allocation sample.
    # The dump also samples jiffies per element, then nft sorts by address.
    # Adjacent remaining lifetimes therefore need not differ by exactly 1 ms.
    # Keep a live, bounded read-back of every configured expires value; the
    # installed member's stricter deadline read and packet checks are separate.
    assert all(0 < expires <= 701+i for i, (_, expires) in enumerate(values)), values
