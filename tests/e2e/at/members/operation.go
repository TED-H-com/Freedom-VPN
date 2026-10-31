package members

import "encoding/json"

// completedOperation reads the operation itself; a successful child step does
// not establish completion of the remaining worker steps.
func completedOperation(body []byte) bool {
	var op struct {
		Status string `json:"status"`
	}
	return json.Unmarshal(body, &op) == nil && op.Status == "succeeded"
}
