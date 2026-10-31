//go:build linux

package streamstats

import (
	"golang.org/x/sys/unix"
	"net"
)

func tcpInfo(c net.Conn) map[string]uint64 {
	tcp, ok := c.(*net.TCPConn)
	if !ok {
		return nil
	}
	raw, e := tcp.SyscallConn()
	if e != nil {
		return nil
	}
	var info *unix.TCPInfo
	if raw.Control(func(fd uintptr) { info, e = unix.GetsockoptTCPInfo(int(fd), unix.IPPROTO_TCP, unix.TCP_INFO) }) != nil || e != nil || info == nil {
		return nil
	}
	return map[string]uint64{"state": uint64(info.State), "rto_us": uint64(info.Rto), "rtt_us": uint64(info.Rtt), "snd_mss": uint64(info.Snd_mss), "rcv_mss": uint64(info.Rcv_mss), "unacked": uint64(info.Unacked), "sacked": uint64(info.Sacked), "lost": uint64(info.Lost), "retrans": uint64(info.Retrans), "total_retrans": uint64(info.Total_retrans), "snd_cwnd": uint64(info.Snd_cwnd), "rcv_space": uint64(info.Rcv_space), "bytes_acked": info.Bytes_acked, "bytes_received": info.Bytes_received, "notsent_bytes": uint64(info.Notsent_bytes), "snd_wnd": uint64(info.Snd_wnd)}
}
