# Diagnostic run eba86d44

AT14/16/17/18/19 passed. AT20 stopped after successful controlled reenrollment because the follow-up 18 documentation required whole usage DB/WAL hash equality. AT15 was not reached. These are the actual records before the harness correction.

Read-only inspection of the quiescent surviving ledger showed D-118 had already run inside ControlledReinstall: current incarnation `01a191b3-5489-7cbb-848d-981d90f9adf3`; origin `01a191b2-30cd-7057-8648-6f8af1b0f606` retained ACK 1 / last sequence 2. Held sequence 2 still had hash `078789b4942a99356384694b3297ef94151d5b68066f85201bea6999d9a832fe`, outbox state, 3278 upload + 4539 download bytes. The new stream had its own sequence 1 and ACK 0. All upload attempts were still held with 503. No SQL writes or local forced rotation occurred.

D-118 necessarily changes the ledger container during adoption. The next run compares every original canonical payload byte and origin sequence/hash/ACK in the retired tables, while retaining the file hash difference as evidence. This does not promote this interrupted run to a pass. Devstack's RESPONSES.md step 4 needs to distinguish preservation of events from file equality once the wrapper starts the new agent; that documentation change is outside this lane.
