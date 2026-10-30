# AT24 offline catch-up and inheritance — follow-up 7

Run `71078ab8-edf8-4d23-af72-ea247f704011`, source `c8d4a00cf1bf594b20016263c2cfc3c164d92bc9`, evidence `2026-10-30T18:09:36Z`. Prefix `w4-updates`, port base 42000.

[Exact snapshot, intent, batch GETs, accepted results and provenance](inheritance.json).

The real primary agent stops; the configured 180-second online
window expires naturally. The new 1.0.2 batch has one offline snapshot member
and reports waiting_online. A replay of the real older success returns its
original ACK without changing the newer job or generation. On reconnect, the
current signed intent moves the job to queued; reporting its target does not
seed over accepted 1.0.1 evidence. The real updater then succeeds and the batch
has exactly one confirmed succeeded target.

Only after that batch exists, shared `Stack.StartSecondExit` and `EnrollAt`
create a second node with a distinct identity and incarnation. It receives the
fleet default through its own system first_sync batch/job and executes the
same shared signed 1.0.2 candidate. Its accepted result and later observation
complete that target. The original batch still contains only the original
node and retains target_count=1. No new node is backfilled into its snapshot.

This uses the existing shared builder, registration and second-node hook.
The second candidate config keeps its generated datapath/resolver values and
sets only the signed release ID/version inputs. No identity is copied and no
projection is seeded. Both nodes run amd64; arm64 artifacts are built but not
executed. The extra pre-existing enrolling-node member from the full §19 AT24
population is outside this restated follow-up.
