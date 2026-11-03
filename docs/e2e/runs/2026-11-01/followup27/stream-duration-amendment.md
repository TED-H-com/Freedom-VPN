# Stream duration amendment, before transport attempt 01

2026-11-01. TEST ONLY. The populated-hold fixture is running; no transport
attempt has started. Keep the eight planned fresh transport stacks (four per
family), and extend each stream from 90 to **150 seconds**, still at 20 ms.
This includes the nominal 120-second rekey interval and a following retry
window. This protocol refinement is not a response to a failed transport row.
The [original plan](plan.md) remains unchanged.

Primary implementation reference: WireGuard's
[messages.h](https://git.zx2c4.com/wireguard-linux/tree/drivers/net/wireguard/messages.h)
defines a five-second retry timeout and a 120-second rekey age; its
[handshake timer](https://git.zx2c4.com/wireguard-linux/tree/drivers/net/wireguard/timers.c)
adds randomized jitter to retransmission. These constants motivate the
measurement window. They do not prove why any particular historical or fresh
packet was delayed, nor identify the exact running kernel's internal timer event.
