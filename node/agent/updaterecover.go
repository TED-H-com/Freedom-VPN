package agent

import "time"

// recoverRetryInterval is the minimum between child starts, including busy
// exits and interruptions. Lock-only deferrals do not spawn a process.
const recoverRetryInterval = time.Second
