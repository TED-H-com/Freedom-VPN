# Established member transports

Follow-up 15 adds `established-transports`: after native admission, both the
persistent TCP echo and authenticated member UDP echo must receive replies,
move that device's Internet counters, then stop those counters after cleanup.
The lease-expiry scenario keeps the same sockets sending across control loss
and requires stable replies/bytes while attempt counts continue increasing.
This is the AT29 active-session fixture; it does not certify every fault or
client platform in §19. See [RUN.md](../RUN.md).
