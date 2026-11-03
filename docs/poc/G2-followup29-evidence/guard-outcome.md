# F27-01 window: failed observer startup, no attribution

The one declared diagnostic attempt is **FAIL**. The NFLOG backend rejected `--time-stamp-precision=nano` before the 100-reload sequence could start:

```text
tcpdump: nflog:29: Can't set nanosecond time stamp precision: That device doesn't support that time stamp precision
```

The [complete command output](guard-window.txt), [receipt](guard-window.json), [captured error](guard-window/TestAT10_AT13_ReloadRevokeBaseline/at10-guard.err), [executed observer source](guard-window/executed/guard_observe.py) and [binary digest](admission/guard-window-binary.json) are retained. The test had made its initial Apply before starting the collectors. No reload sequence, guard verdict records, exact drop attribution or diagnostic D-133 timing distribution was obtained. The early-startup failure did not retain that initial Apply sidecar; it is not represented as a measured cost sample. F27-01 stays unresolved; B1 stays FAIL. The attempt is spent, and **no retry was made**.

Admission was enforced: the reservation was acquired in 0.035 seconds, followed by 60 seconds settling. The first two 30-second checks rejected an active gate process and runnable-count violations; the third passed at 210.486 seconds. A second preflight after topology startup rejected one runnable-count violation, then passed at 120.284 seconds. The [competing-process record](guard-window-competing-process.json) places that process's start one second after reservation acquisition; it could have passed admission immediately before the reservation. No claim is made that the reservation alone prevents that race. Both accepted samples had zero actual competing processes. All preflights remain in [before-startup](admission/guard-window-before.json) and [after-startup](admission/guard-window-after-startup.json). The [post-run check](guard-window-posthost.json) found no active competitor, and the [release](guard-window-release.json) succeeded. [Cleanup](guard-cleanup-inventory.json) lists zero owned containers, images, networks or volumes.

## Offline correction, not a second measurement

The current observer uses the backend's default pcap precision and decodes both microsecond and nanosecond little-endian captures, retaining `pcap_precision_ns`. A microsecond fixture [fails against the executed parser](guard-micro-before.txt) and [passes after correction](guard-micro-after.txt). The original nanosecond fixture still passes. This correction has **not** been verified against live NFLOG because that would consume another diagnostic window. It does not supply missing guard evidence, and later ordinary gates leave the observer disabled.

The production renderer/install path is unchanged. The proposed TEST ONLY rule observation and its cost limits remain in the [predeclaration](guard-declaration.md). A new authorization/window is required to validate the collector, obtain exact drop records and measure its diagnostic cost. Any proposal to ship rule logging or alter the production guard also requires a separate D-133 comparison. D-120 and all historical statuses are unchanged.
