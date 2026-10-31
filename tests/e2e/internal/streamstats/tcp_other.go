//go:build !linux

package streamstats

import "net"

func tcpInfo(net.Conn) map[string]uint64 { return nil }
