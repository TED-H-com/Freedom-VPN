package nativecapture

import "testing"

func TestIPv4CounterPairs(t *testing.T) {
	got := SNMP4Counters([]byte("Ip: InReceives InHdrErrors InAddrErrors\nIp: 42 2 3\nUdp: InErrors NoPorts\nUdp: 4 5\nTcp: MaxConn\nTcp: -1\n"))
	if got["IpInHdrErrors"] != "2" || got["IpInAddrErrors"] != "3" || got["UdpInErrors"] != "4" || got["UdpNoPorts"] != "5" || got["TcpMaxConn"] != "" {
		t.Fatalf("bad counters: %v", got)
	}
	if len(SNMP4Counters([]byte("Ip: InReceives InHdrErrors\nIp: 42\n"))) != 0 {
		t.Fatal("malformed counter pair accepted")
	}
}
