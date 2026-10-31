package members

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
)

type selectedPath struct {
	Connected bool   `json:"connected"`
	Type      string `json:"connection_type"`
	Family    string `json:"chosen_family"`
}

// Match the selected exit in memory; never export its peer id, key or endpoint.
func selectNativePath(raw []byte, target string) (selectedPath, error) {
	var status struct {
		Peers struct {
			Details []struct {
				IP       string `json:"netbirdIp"`
				Status   string `json:"status"`
				Type     string `json:"connectionType"`
				Endpoint struct {
					Remote string `json:"remote"`
				} `json:"iceCandidateEndpoint"`
			} `json:"details"`
		} `json:"peers"`
	}
	if json.Unmarshal(raw, &status) != nil {
		return selectedPath{}, fmt.Errorf("native status malformed")
	}
	for _, peer := range status.Peers.Details {
		if strings.Split(peer.IP, "/")[0] != strings.Split(target, "/")[0] {
			continue
		}
		out := selectedPath{Connected: strings.EqualFold(peer.Status, "Connected"), Type: peer.Type, Family: "unavailable"}
		host, _, err := net.SplitHostPort(peer.Endpoint.Remote)
		if err == nil {
			if ip := net.ParseIP(host); ip != nil {
				out.Family = "ipv6"
				if ip.To4() != nil {
					out.Family = "ipv4"
				}
			}
		}
		return out, nil
	}
	return selectedPath{}, fmt.Errorf("selected exit absent from native status")
}
