# installer-wrappers

RerunInstaller and ControlledReinstall on the systemd installer node

Status: fail

Run: 299afccc-d98b-4bde-821e-6930bdbbfa69

Source: 9e493e8f6407dd7f091963fb7e133a11123305c3

Spec: §6.3, ENR-06

- container: fvpn-e2e-installer-node
- regular_node: unchanged
- exit_64_argv: 64
- exit_64_mode: 64
- install_exit: 1
- install_status: -
- install_incarnation: 01a19012-d25f-75c8-b452-fcefa52dfe43
- install_message: failed: starting fvpn-node-netbird-up.service failed: p from the setup key file... 2026-10-31T03:48:46.893Z WARN client/internal/profilemanager/profilemanager.go:122: failed to get config directory: neither $XDG_CONFIG_HOME nor $HOME are de
