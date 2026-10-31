# Follow-up 14 acceptance

Run: 4e3e9487-fe0f-4d88-a396-d5d5fa4a8e1d

Source: 7029dbe631b8344ccd0db3991fac81e773a26ed5

Started: 2026-10-31T00:42:10Z (stack-health record)

Result: pass=29 blocked=0 fail=0. Go test 855.85s. make -f mk/e2e.mk e2e-test exit 0.

Exports, fresh stack, `FVPN_E2E_EXPORTS=1`, `TestE2E_HarnessExports`: pass=16 blocked=0 fail=0. Go test 273.12s. Run `0a687f3a-2351-4234-b43d-882109267967`. Same source. Records are `exports/`.

There is no e2e-build target. e2e-up built the images.

Image digests (manifest lists from this up):

fvpn-e2e-node:run sha256:a73f98e04bce4a2e1b9a575a7ecb641a1e76d18ba8bb93749dbc968a9b424226
fvpn-devstack-node:local sha256:41b4388750ef87ee1e8e22aba171ab156bb16df1da220553c5a5a9573dfc2235
fvpn-devstack-worker:local sha256:47b81661daa6f6893cad8dbc16f0d11f22da8393257ed2a7a63df6668e2ac8f6
fvpn-devstack-control:local sha256:cfb96063da763089e9e459a3a66b5b989b2ac6a86a07df4aa2d438bcee850176
fvpn-devstack-origin:local sha256:310819d08da9eb18dd2be7b5456aaeb0b6b9ff1ab3175026ea2edd764d96fa0a
fvpn-devstack-payment:local sha256:ab0f9d728b5357d30f92a5ec5abd3ed1c73e36b75ae3e3b44c20458ce1180c91
fvpn-devstack-idp:local sha256:10c5d2531733d0b0d39bd5962f07e5bc86e560f2b8a2a8b0e8ce77090f42d6d5
fvpn-e2e-worker:run sha256:4695c0961ecad94aa6f8f6f230c6c2b5674337a1acb9c20c551918c839783274
fvpn-e2e-control:run sha256:ca987b86fb2d1afcd8369604a1eb8680e3bfd28ad3f3091dbd3048e9776e6c4f
fvpn-e2e-origin:run sha256:8282e13ed70b661195454797600dbcffb675a1cd8299b15efc7b850947cb42f5
fvpn-e2e-payment:run sha256:f9b7b12bf20040c86b028c835896026407b8f4f529fb5720b297f0c707632c18
fvpn-e2e-idp:run sha256:61eddd0d85d2731d12b0d7eb4b5476cd0f173aa2c08ff5320ee647f207f60d39
fvpn-devstack-postgres:17 sha256:eefea889afa22681cba283ecfdab510f2717b774aef84aa8a37fbb4fee2e4f3f
fvpn-e2e-postgres:run sha256:50a504bef07921705771ef48333a73540519b25eeec69317d1b5b7443eb1a5b5
fvpn-devstack-netbird:0.82.0 <none>
fvpn-devstack-nginx:1.30.0 <none>
fvpn-devstack-adguard:v0.107.59 sha256:5d5e3aef39a8c8906a324f0c2da0a6123929e459b57c1e3f05e401dca5c7bf02
fvpn-devstack-coredns:1.12.0 sha256:40384aa1f5ea6bfdc77997d243aec73da05f27aed0c5e9d65bfa98933c519d97
