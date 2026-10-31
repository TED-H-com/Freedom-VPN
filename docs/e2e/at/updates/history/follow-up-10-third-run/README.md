# Follow-up 10 third diagnostic run

Run `cf1a8c61-aced-4ab3-aa7e-a28cca1716e9`, harness `a6403f296bf38d77fe01ca912868d3b559608bfb`, product main `fb5e002549f79e8ef61bebcf42cce1678ab65cb1`. Copied byte-identically with SHA-256 pins. Installer extension passes; regular AT23 candidate exits with `agent: active release identity is inconsistent`, then no last-good rollback target exists. AT24–AT27 remain blocked. Go test exits 1 (259.370 s), make exits 2.

The raw probe narrative in this diagnostic still uses an unconditional historical success description even though its measured booleans are false. The next harness revision makes those descriptions conditional and captures the differing executable paths. No probe traversal or regular AT pass is claimed here.
