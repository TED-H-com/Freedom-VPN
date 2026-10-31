# Follow-up 7 attempt 2 — live meter sequence collision

Run `c8939ad8-ec4f-413a-aac3-1a4bf5d35b32`, source `fd7843de0d4d99d9c926d854a3db009bcd9271d6`, clean at start; product base `d9e28f8`.

All D-077 checks in AT02 and AT03 passed, including D-081's 409 with fence retained, explicit re-grant and counted traffic. AT02's inherited synthetic positive usage probe failed: it selected `max(event_seq)+1 = 10` while the live agent still owned that sequence. The agent submitted sequences 10/11 at `2026-10-31T03:58:41.305327Z`; both synthetic submissions then returned `conflict`. The existing seq-10 receipt had one sample and one conflict row. This is a harness allocation error; the usage handler correctly refuses different payloads for an existing sequence.

The foreground suite was interrupted with SIGINT (exit 130) during AT04. Completed and partial records are preserved without relabeling; no pass is carried into the final baseline. The stack is reset because this diagnostic created a durable conflict.

Correction: the positive probe reads a real accepted event's canonical payload, verifies its stored hash, resends it twice over node mTLS and requires two `duplicate` responses with unchanged receipt/sample counts. It never allocates a positive event sequence. The unknown-device negative remains: the shared supervisor stops the agent before choosing its rejected probe sequence and resumes the same selected release in cleanup. This diagnostic runs after the uninterrupted-agent revoke/re-grant traffic assertions.

Recorded cases: [('AT01', 'blocked', True), ('AT02', 'fail', True), ('AT03', 'pass', True), ('AT04', 'blocked', False), ('AT05', 'blocked', True)]
