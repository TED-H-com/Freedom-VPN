# Node PKI, enrollment and signing

These are Go domain services; HTTP authorization, middleware and NetBird jobs
remain separate. The contract is `contracts/openapi.yaml`; this implementation
uses its generated request/response types. References below are to Project Spec
v1.0, `docs/spec/PROJECT_SPEC.zh-TW.md`.

## Trust boundaries (§2, §6, §13.4, §16.1–2)

| Purpose | Algorithm | Private material | Consumer |
|---|---|---|---|
| `desired-state` | Ed25519 | Central provider | Node policy verifier |
| `lease` | Ed25519 | Separate central key | Node lease verifier |
| `recovery` | Ed25519 | Separate central key | Node checkpoint verifier |
| `ledger-epoch` | Ed25519 | Separate central key | Central recovery code |
| `node-ca` | ECDSA P-256 | Provider, through a guarded `crypto.Signer` | PKI service |
| `bootstrap-secret` | AES-256-GCM | Provider | Temporary NetBird secret storage |
| `release` | Ed25519 verification only | **Never accepted by this service** | Central and node release verifiers |

`signing.ForPurpose` returns a `Signer` bound to one central purpose. Providers
produce `pkg/envelope.Envelope`; the development provider uses `envelope.Sign`.
This interface permits a later KMS/HSM provider without exporting private keys
or changing consumers. Signatures and verification use the existing envelope
rules, including rejection of any bad known-key signature. Consumers still
validate audience, node/incarnation, schema, time and generation of the signed
document; this generic key service does not replace those domain checks.

`FileProvider` is a **development provider**, with 0700 directory and 0600 files,
atomic replacement plus fsync and cross-process locking. A fresh directory must
be explicitly initialized. Opening a missing/corrupt state ledger fails closed.
It permits one active signer per central purpose, retires old signers on rotation,
and retains public keys and revocation tombstones. Its ledger is outside
PostgreSQL, and must remain outside PostgreSQL snapshot restoration. Revoked
CA handles recheck the ledger on every signature, even if previously acquired.
Nodes receive only `TrustBundle`, serialized in the `pkg/envelope` trust-store
format. It excludes CA private material, encryption keys and central-only
ledger-epoch keys. See [rotation and recovery](rotation.md) for the limits of a
local file ledger and production requirements.

## CA and identity (§6.4, §14.4, §15.3)

CSRs must be PKCS#10 PEM, ECDSA P-256/SHA-256, correctly self-signed, with an
empty subject and no requested extensions or attributes. Personal names,
hostnames, email, challenge passwords and caller-selected SANs are rejected.
Input is bounded to 8192 bytes. A node's private key is never an API input.

Client certificates have digital-signature usage and client-auth EKU only.
The CA supplies exactly these two URI SANs, using lowercase hyphenated UUIDs:

```text
urn:fvpn:node:<node_id>
urn:fvpn:incarnation:<incarnation_id>
```

Serials are nonzero random 128-bit integers, encoded as lowercase hexadecimal.
Issuer identity is SHA-256 of the issuing CA certificate DER. The database
enforces `UNIQUE(issuer,serial)` and retains certificate DER, fingerprints,
validity and revocation metadata, never a node or CA private key.

`pki.Service.Authenticate(ctx, cert)` verifies DER against the configured
self-signed P-256 node CA, client EKU, the exact SAN identities, local durable
revocation journal, registered certificate fingerprint, lifecycle, active
incarnation, revocation and effective validity. It reparses DER so mutated
fields on a caller's `x509.Certificate` cannot change the signed identity.
Publication `disabled`/`unpublished` does not remove control access. Revoked or
retired identities are refused. Database or revocation-journal failure denies
access. TLS must independently prove possession of the certificate private key.

`Renew` accepts the authenticated TLS certificate and the generated
`CertificateRenewRequest`. Body identity fields are consistency checks, never
selectors. A new verified CSR gets a new serial in the same incarnation.
`renewal_request_id` plus canonical request hash deduplicates retries. All older
serials get a database `usable_until` cap; repeated renewal cannot extend it.
The original incarnation fingerprint stays as enrollment history. Renewal is
the path for rotating the device key; an old enrollment replay cannot restore
an expired or superseded certificate.

CA signing happens before request transactions. The transaction rechecks
identity while holding the node row lock before storing the certificate, so
revocation wins over in-flight renewal. The local revocation journal is read
under the lock too; it must not be a network-backed implementation.

## Configuration

| Setting | Behavior |
|---|---|
| `InstallerValidity` | Configurable default 15 minutes |
| `PreinstalledValidity` | Configurable default 7 days |
| Create request `expires_in_seconds` | Explicit override, 60–2592000 seconds |
| CA certificate validity | Explicit provisioning input; production value 待決策 |
| Node certificate validity | Required `NewCA` parameter; `TestCertificateValidity=24h` is **TEST ONLY**, production 待決策 |
| Renewal overlap | Required `NewService` parameter, less than certificate validity; `TestRenewalOverlap=5m` is **TEST ONLY**, production 待決策 |
| Bootstrap delivery validity | Required config; `TestBootstrapValidity=10m` is **TEST ONLY**, production 待決策 |
| Signing validity/overlap | Required `Rotate` arguments; production values 待決策 |
| Endpoint active window | Required `ObserveEndpoint` argument; production policy 待決策 |

NetBird provisioning requests `one_off=true`, `usage_limit=1`,
`ephemeral=false`, staging group role, and a 86400-second external setup-key
expiry (the §6.4 minimum). That external lifetime is independent of the shorter
local bootstrap-delivery deadline. The worker must resolve the configured
staging group; no real group/account identifier is embedded here.

## Service integration

1. Apply embedded migration `00110_pki_enrollment.sql` with the existing DB tool.
2. Initialize/open a provider outside database backup restoration; provision
   separate purpose keys and a public CA certificate. Inject it into `pki.CA`,
   `pki.Service` and `enrollment.Service`; use explicit production time settings.
3. Admin adapters enforce MFA/session/CSRF, derive `Command.Principal` from the
   session, parse UUID idempotency keys, and use `apierr.Write` on errors.
   `RevokeNode` also requires the parsed `If-Match` resource version.
4. Agent adapters take the peer certificate from verified mTLS; never trust a
   body-supplied certificate, `X-Node-ID` or an unprotected proxy header.
   `Authenticate` locally returns `NodeIdentity{NodeID, IncarnationID}`. `AuthenticateAgent` adapts it to `internal/auth.AgentAuthenticator`.
5. `CreateEnrollment` returns `api.EnrollmentCreated`; `Enroll` returns
   `EnrollResult.Response`, 201 for first success and 200 when `Replay` is true.
   Every error path returns no partial secret/result. Secret responses require
   `Cache-Control: no-store`; middleware must redact Authorization/body secrets.
6. Workers consume `node.provision.requested` and persist external read-back
   evidence before calling `StoreBootstrap`. They must not mark the operation
   successful until setup/peer/policy required steps have real evidence.
   Call `ExpireBootstrap` periodically. `TakeBootstrap` requires the TLS
   certificate and commits erasure before returning plaintext.
7. Revocation and retirement enqueue one `node.publication_changed` event for
   the exits executor, with standard withdrawal steps plus required
   `netbird_identity_cleanup`. The worker confirms route removal, setup-key
   revocation and peer deletion before finalizing; an offline agent remains
   unconfirmed. `RevokeNode` returns a pending operation, never external completion.

The D-015 `POST /agent/bootstrap` contract now carries authenticated one-time
setup delivery. Reinstall accepts an existing `node_id` only after the current
incarnation is revoked; a retired Node stays terminal. The embedded domain
migration `00111_pki_reinstall.sql` enables this guarded transition. The base F-16
guard already permits the first deployment/peer binding after NetBird read-back.

See [node wiring](../nodes/WIRING.md) for the mounted APIs, worker configuration,
current integration evidence and remaining cross-area blockers. Node trust
bundles now use `envelope.NewNodeTrustStore` and exclude ledger-epoch keys.

## Verification and scope

```sh
make -f mk/pki.mk pki-verify
make -f mk/pki.mk pki-test-integration
make go-lint go-test
```

Integration tests require `FVPN_TEST_DATABASE_URL`; `dbtest` creates and drops
isolated databases. `pki-test-integration` runs without cached test results.
All test scratch is disk-backed under `.agent/tmp`, never
tmpfs. `pki-build` cross-compiles both amd64 and arm64. No external network,
NetBird instance, root access or container is needed for unit tests.

Tests cover CSR/chain restrictions, proof vectors, rotation/terminal revocation,
token races/replay, atomic failure rollback, certificate renewal and overlap,
authenticated secret delivery/erasure, endpoint conflicts, and scans of all
table columns plus captured logs. AT05/AT07/AT28 are service-layer evidence;
real devices, image preparation, installer preservation, NetBird read-back and
HTTP role enforcement remain with the owning tasks and are not claimed tested.
