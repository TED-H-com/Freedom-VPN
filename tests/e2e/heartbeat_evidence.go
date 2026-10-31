package e2e

import (
	"strconv"
	"strings"
	"time"
)

// A retained heartbeat cannot prove service restoration. Require a new wall
// timestamp from this shared-clock lab as well as serving=1.
func servingHeartbeatSince(raw []byte, since time.Time) bool {
	fields := strings.Fields(string(raw))
	serving := false
	var at int64
	for i := 0; i+1 < len(fields); i++ {
		switch fields[i] {
		case "serving":
			serving = fields[i+1] == "1"
		case "wall_unix_ns":
			at, _ = strconv.ParseInt(fields[i+1], 10, 64)
		}
	}
	return serving && at > 0 && !time.Unix(0, at).Before(since)
}
