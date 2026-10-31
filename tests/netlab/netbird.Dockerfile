# TEST ONLY: same pinned client/server as deploy/dev/netbird. No cloud access.
ARG LAB_IMAGE=fvpn-w1-netlab-real:lab
FROM netbirdio/netbird:0.82.0@sha256:eca5b274c635bfbb04ec6724a0712d1146190dd450eea199eeb83916fdd94965 AS client
FROM netbirdio/netbird-server:0.82.0@sha256:7b5b53b2dc8bc25639f791cf72d47cf86911894ff3f8a97cb79bab9be958109c AS server
FROM ${LAB_IMAGE}
COPY --from=client /usr/local/bin/netbird /usr/local/bin/netbird
COPY --from=server /go/bin/netbird-server /usr/local/bin/netbird-server
