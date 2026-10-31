# Forced relay

`forced-relay` reads connected NetBird paths, applies the bounded member
underlay UDP fault, and requires a relayed path with no direct peers. TCP and
authenticated member UDP must work there. After removal it requires direct
path restoration and continued replies on those same sockets. The drop match
expires after 180 seconds; normal cleanup deletes the owned member table.
This supplies an AT31 path fixture, not a relay capacity certification.
