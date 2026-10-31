# Follow-up 5 diagnostic run 014daad8

Harness/image `565031bf`. AT14, AT16, AT17, AT18 and AT19 passed; AT15 was not reached. AT20 delivered the surviving tail through production D-118, retained its original attribution, and proved 33720 + 7823 = 41543 member bytes, then failed an overly strict raw HTTP/storage byte comparison.

[The same-stack replay diagnostic](replay-format-diagnostic.json) records identical stored/replayed JSON values, including the original timestamp, accepted status and origin ACK 2. Storage contains canonical JSON (323 bytes); HTTP encoding adds a newline (324 bytes). The harness now compares the full canonical HTTP result against the stored canonical bytes, following the central wire test. It retains the raw-byte comparison separately. No product change or failure-case pass is inferred. The failed AT20 record remains unchanged here.
