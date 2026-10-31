# installer-wrappers

RerunInstaller and ControlledReinstall on the systemd installer node

Status: fail

Run: 12d10041-9d33-4fbb-a4b0-9d2a21bce785

Source: 4fb5fa88ac400605ac08ee42c8eac99e936ec2a3

Spec: §6.3, ENR-06

- container: fvpn-e2e-installer-node
- regular_node: unchanged
- exit_64_argv: 64
- exit_64_mode: 64
- install_exit: 1
- install_status: -
- install_incarnation: 01a19039-037e-7453-9c6d-8109b3c2a549
- install_code: failed
- owner: packaging/installer + deploy/node/systemd: real NetBird service startup
- netbird_socket_readonly: true
- agent_heartbeat_readonly: true
- rerun_reenroll_success: not reached: initial install failed; no service override or NetBird stub
- install_message: failed: starting fvpn-node-netbird-up.service failed: the setup key file... 2026-10-31T04:30:30.496Z WARN client/internal/profilemanager/profilemanager.go:122: failed to get config directory: neither $XDG_CONFIG_HOME nor $HOME are defined f
