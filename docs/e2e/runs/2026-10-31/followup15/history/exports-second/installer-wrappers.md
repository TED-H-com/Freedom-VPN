# installer-wrappers

RerunInstaller and ControlledReinstall on the systemd installer node

Status: fail

Run: 630b9f07-62d2-4db1-9737-9060732db1f1

Source: 3c25ce9daa081874988c9b09bc8c26e66d3c1b46

Spec: §6.3, ENR-06

- container: fvpn-e2e-installer-node
- regular_node: unchanged
- exit_64_argv: 64
- exit_64_mode: 64
- install_exit: 1
- install_status: -
- install_incarnation: 01a19022-0cf7-7719-88de-909e0c3818ed
- install_code: failed
- owner: packaging/installer + deploy/node/systemd: real NetBird service startup
- netbird_socket_readonly: true
- agent_heartbeat_readonly: true
- rerun_reenroll_success: not reached: initial install failed; no service override or NetBird stub
- install_message: failed: starting fvpn-node-netbird-up.service failed: p from the setup key file... 2026-10-31T04:05:35.150Z WARN client/internal/profilemanager/profilemanager.go:122: failed to get config directory: neither $XDG_CONFIG_HOME nor $HOME are de
