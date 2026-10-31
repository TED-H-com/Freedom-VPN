# installer-wrappers

RerunInstaller and ControlledReinstall on the systemd installer node

Status: fail

Run: 926791b9-36ef-4051-9d44-cd7eafdc8792

Source: 9e493e8f6407dd7f091963fb7e133a11123305c3

Spec: §6.3, ENR-06

- container: fvpn-e2e-installer-node
- regular_node: unchanged
- exit_64_argv: 64
- exit_64_mode: 64
- install_exit: 1
- install_status: -
- install_incarnation: -
- install_message: Created symlink /etc/systemd/system/multi-user.target.wants/fvpn-watchdog.service → /etc/systemd/system/fvpn-watchdog.service.
Created symlink /etc/systemd/system/multi-user.target.wants/fvpn-agent.service → /etc/systemd/system/fvpn-age
