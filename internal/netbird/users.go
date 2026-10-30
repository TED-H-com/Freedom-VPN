package netbird

import (
	"context"
	"net/http"
	"strings"
)

// User is the identity metadata needed for native-peer discovery (§15.2).
// Email, display name and other provider profile data are deliberately omitted.
type User struct {
	ID          string
	ServiceUser bool
	Blocked     bool
}

// ListUsers reuses the authenticated client's timeout, retry and response bounds.
// It observes provider identities; the caller proves their link to local members.
func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	var response []struct {
		ID          string `json:"id"`
		ServiceUser *bool  `json:"is_service_user"`
		Blocked     *bool  `json:"is_blocked"`
	}
	if err := c.call(ctx, http.MethodGet, endpoint("users", ""), nil, &response); err != nil {
		return nil, err
	}
	if response == nil {
		return nil, failure(Invalid)
	}
	users := make([]User, 0, len(response))
	seen := make(map[string]bool, len(response))
	for _, user := range response {
		if !valid(user.ID) || strings.TrimSpace(user.ID) == "" || user.ServiceUser == nil || user.Blocked == nil || seen[user.ID] {
			return nil, failure(Invalid)
		}
		seen[user.ID] = true
		users = append(users, User{ID: user.ID, ServiceUser: *user.ServiceUser, Blocked: *user.Blocked})
	}
	return users, nil
}
