package isolation

import (
	"regexp"
	"strconv"
	"strings"
)

var captureCount = regexp.MustCompile(`(?m)^(\d+) packets? captured$`)
var captureNoLoss = regexp.MustCompile(`(?m)^0 packets dropped by kernel$`)

// CaptureCount accepts tcpdump's singular and plural summaries. A missing
// startup/summary or kernel loss cannot establish an empty ingress capture.
func CaptureCount(raw string) (int, bool) {
	m := captureCount.FindStringSubmatch(raw)
	if len(m) != 2 || !strings.Contains(raw, "listening on") || !captureNoLoss.MatchString(raw) || strings.Contains(raw, "pcap_loop:") {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil
}
