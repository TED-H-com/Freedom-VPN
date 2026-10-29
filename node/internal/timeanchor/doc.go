// Package timeanchor keeps the node's trusted clock (§16.3).
//
// A signed server time plus a monotonic elapsed reading is the only clock
// that may move a deadline. Wall-clock steps, including a jump backward,
// never lengthen a lease. After reboot, a lost anchor or a clock anomaly,
// member forwarding stays off until a fresh lease and a new anchor arrive.
package timeanchor
