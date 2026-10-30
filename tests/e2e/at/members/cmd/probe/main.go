// Command probe is a bounded acceptance probe copied only into this task's lab.
// It prints aggregate DNS answers; HTTP credentials arrive on stdin, never argv.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspolicy"
	"golang.org/x/net/dns/dnsmessage"
)

func main() {
	if len(os.Args) < 3 {
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	switch os.Args[1] {
	case "transfer":
		transfer()
	case "observe":
		observe()
	case "dns":
		if len(os.Args) != 6 || (os.Args[2] != "127.0.0.1:53" && os.Args[2] != "172.28.31.40:53" && os.Args[2] != "172.28.31.54:53") {
			os.Exit(2)
		}
		typ, err := strconv.ParseUint(os.Args[4], 10, 16)
		if err != nil {
			os.Exit(2)
		}
		res, err := dnspolicy.NewDNSClient(2*time.Second).QueryProto(ctx, os.Args[5], os.Args[2], os.Args[3], dnsmessage.Type(typ))
		if err != nil {
			fmt.Println(`{"available":false}`)
			return
		}
		ttl := res.AuthorityTTL
		if len(res.Answers) > 0 {
			ttl = res.Answers[0].TTL
		}
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"available": true, "rcode": int(res.RCode), "answers": len(res.Answers), "ttl": ttl})
	case "http":
		allowed := map[string]bool{"/control/querylog/config": true, "/control/querylog": true, "/control/stats/config": true, "/control/stats": true, "/control/filtering/status": true, "/control/dns_info": true}
		endpoint := "http://127.0.0.1:8080" + os.Args[2]
		if os.Args[2] == "metrics" {
			endpoint = "http://127.0.0.1:9091/metrics"
		} else if !allowed[os.Args[2]] {
			os.Exit(2)
		}
		password, _ := io.ReadAll(io.LimitReader(os.Stdin, 4096))
		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			os.Exit(2)
		}
		if len(password) > 0 {
			req.SetBasicAuth("e2e", strings.TrimSpace(string(password)))
		}
		res, err := (&http.Client{Timeout: 4 * time.Second}).Do(req)
		if err != nil {
			os.Exit(1)
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			os.Exit(1)
		}
		_, _ = io.Copy(os.Stdout, io.LimitReader(res.Body, 2<<20))
	default:
		os.Exit(2)
	}
}
