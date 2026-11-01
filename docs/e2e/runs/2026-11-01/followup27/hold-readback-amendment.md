# Hold fixture readback correction and declared second attempt

2026-11-01, after hold attempt 01 and before attempt 02. Attempt 01 formed a
two-member hold for 49.985 seconds, reported the Step-3 alarm, escalated once
after 9.886 seconds, and ended `generation_matched`. Its final runner verdict
is **failed**: the extra byte-equality assertion tried reading root-owned live
SQLite state from the unprivileged host and returned without a receipt.
That is a fixture readback bug, not a demonstrated product defect.

Keep attempt 01 unchanged. Add a content-free in-container read-only SQLite
helper; do not chmod the store, stop the agent or copy a live DB/WAL pair.
Retain a separate readback supplement on attempt 01, then run one additional
fresh hold fixture (attempt 02) with the corrected assertion. This second
attempt proves the complete runner; it does not replace the failed first row
or exercise the consumer acceptance matrix. Transport counts stay eight.
