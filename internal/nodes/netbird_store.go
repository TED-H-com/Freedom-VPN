package nodes

import "github.com/TED-H-com/Freedom-VPN/internal/devices"

// NetBirdBindings shares the deployment lock and durable binding encoding with
// every member/device adapter writer (D-026). Keep one implementation of both.
type NetBirdBindings = devices.NetBirdBindings
