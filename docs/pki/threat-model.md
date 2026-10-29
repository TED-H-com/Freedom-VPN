# Threat model

References: §6.2, §14.4–7, §16.1–2 and §17.4. The trusted components are the TLS
terminator, authenticated admin adapter, service process, current CA/provider,
current revocation journal and database. A reverse proxy must have a protected
backend channel and block direct/header-spoofed access. Certificate possession
is established by TLS; a certificate parsed from a body is not authentication.

| Attacker capability | What it permits | What it does not permit |
|---|---|---|
| Stolen unused token | Race to enroll an attacker-generated key before expiry/revocation | Read central keys, choose another reserved node, publish an exit, bypass staging/policy checks |
| Stolen consumed token alone | Present a rejected replay attempt | Read the certificate or bootstrap secret without valid CSR-key proof; replace the binding |
| Consumed token plus a captured complete valid enrollment request | Replay that exact proof before token expiry and recover the already-public certificate | Authenticate mTLS without the node private key; retrieve setup-key plaintext through enroll |
| Stolen public client certificate | Learn its public node/incarnation IDs and key | Prove private-key possession in TLS or renew by itself |
| Stolen certificate **and private key** | Impersonate that node while serial/identity remain valid; possibly claim its undelivered bootstrap or renew | Select another node through body IDs; gain admin/member roles; continue after revocation |
| Stolen database copy | Read public certificates, metadata, token hashes and encrypted pending bootstrap material | Recover a 256-bit random token by feasible guessing or decrypt AES-GCM material without its independent provider key |
| Old PostgreSQL snapshot restored with current provider/journal | Reintroduce stale domain data, requiring restore quarantine and reconciliation | Reactivate a retired/revoked signing key or authenticate a node denied in the current journal |
| Old database **and** old provider/journal restored together | Potentially revive old authorization; this is outside FileProvider's rollback protection | Be treated as a verified-current authority without independent recovery evidence |
| Compromised central service or provider host | Use its permitted central keys and access pending plaintext during delivery | Obtain a release signing key from this provider; it never contains one |

Token possession never substitutes for a CSR proof. A captured proof is
replayable by design; it gives public recovery material, not TLS possession or
a new identity. Claiming a fingerprint without verifying the actual CSR and
proof is not accepted. Both CSR and proof are ECDSA P-256/SHA-256; the proof
format is the current contract and carries no new hidden challenge protocol.

Node revocation is terminal in database guards and the independent journal.
File journal writes precede commit; a DB failure can deny access before the
withdrawal intent commits, so operations must retry the command. This is an
availability cost chosen to prevent stale restored identity authorization.
Already signed offline documents are bounded by their own expiry and verifier
revocation distribution. Revoking a certificate does not by itself prove that
NetBird routes/peers are removed: the returned operation stays pending until
the worker records read-back evidence (§5.2).

Setup-key ciphertext is authenticated to node/incarnation/operation/external
ID/expiry, then erased atomically on first delivery. A response lost after
commit loses the one-time delivery; remote reconciliation is required. A
database restore can restore old ciphertext or an unconsumed token row. Those
receipts are not claimed protected by the signing ledger: the restore runbook
requires quarantine and reconciliation before authorization is reopened.

Privacy controls: endpoint conflict tracking stores only compatibility hashes,
not network destination lists. Service logs contain fixed messages and resource
UUIDs. Errors returned from services contain fixed public messages, not SQL,
CSR, certificates, tokens, private keys or provider responses. Secret-bearing
responses must be `no-store`; the HTTP adapter and its access logs are part of
the remaining boundary. Integration tests scan every table column for generated
tokens/setup secrets and private-key encodings, and capture the service logs.

Out of scope here: request rate limits/CSRF/OIDC/MFA enforcement in the HTTP
framework, node-side key permissions and image preparation, NetBird API
reconciliation, production KMS/HSM integration, distributed vault replication,
and real hardware/network evidence. No remote shell, endpoint MDM, TLS
interception or source-specific trust tier is introduced.
