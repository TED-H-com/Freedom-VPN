//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (r *runner) establishedTransports(ctx context.Context) {
	rec := Record{ID: "established-transports", Title: "Persistent member TCP and authenticated UDP echo", Spec: []string{"§8", "AT29"}, Status: StatusFail}
	note := recPathNote(&rec)
	defer func() { r.record(rec) }()
	s := &Stack{lab: r.lab}
	defer func() { _ = s.StopEstablishedTransports(context.Background()) }()
	if r.usageDevice == uuid.Nil || !r.flowCounted {
		rec.Status = StatusBlocked
		note("blocker", "counted member admission is the prerequisite")
		return
	}
	before, err := s.DeviceByteCounters(ctx, r.usageDevice)
	if err != nil || !before.Present {
		rec.Status = StatusBlocked
		note("blocker", "device counters are not in the ruleset")
		return
	}
	note("bytes_before", strconv.FormatUint(before.Upload+before.Download, 10))
	if _, err = s.StartEstablishedTransports(ctx, r.usageDevice); err != nil {
		note("start", err.Error())
		note("owner", "w3-devstack-e2e: member echo helper")
		return
	}
	wait, cancel := context.WithTimeout(ctx, 20*time.Second)
	var mid DeviceCounters
	moved := waitUntil(wait, time.Second, func() bool {
		mid, err = s.DeviceByteCounters(wait, r.usageDevice)
		return err == nil && mid.Present && mid.Upload+mid.Download > before.Upload+before.Download
	})
	cancel()
	tcpN, udpN, problem, statusErr := s.MemberSessionStatus(ctx)
	note("tcp_echoes", itoa(tcpN))
	note("udp_replies", itoa(udpN))
	if problem != "" {
		note("session", problem)
	}
	if statusErr != nil {
		note("session_status", "unavailable")
	}
	if !moved {
		note("bytes_mid", "not increased")
		note("owner", "w3-devstack-e2e: established session did not move device counters")
		return
	}
	note("bytes_mid", strconv.FormatUint(mid.Upload+mid.Download, 10))
	if err = s.StopEstablishedTransports(ctx); err != nil {
		note("stop", err.Error())
		return
	}
	stopped, err := s.DeviceByteCounters(ctx, r.usageDevice)
	if err != nil || !stopped.Present {
		note("bytes_stopped", "unavailable")
		return
	}
	time.Sleep(2 * time.Second)
	later, err := s.DeviceByteCounters(ctx, r.usageDevice)
	if err != nil || !later.Present {
		note("bytes_after", "unavailable")
		return
	}
	note("bytes_stopped", strconv.FormatUint(stopped.Upload+stopped.Download, 10))
	note("bytes_after", strconv.FormatUint(later.Upload+later.Download, 10))
	if later.Upload != stopped.Upload || later.Download != stopped.Download {
		note("owner", "w3-devstack-e2e: counters kept moving after the session stopped")
		return
	}
	if tcpN < 1 || udpN < 1 {
		note("owner", "w3-devstack-e2e: TCP echo and authenticated UDP echo both have to complete")
		return
	}
	note("probe", "updater UDP 4443 was not used")
	rec.Status = StatusPass
}

func (r *runner) forcedRelay(ctx context.Context) {
	rec := Record{ID: "forced-relay", Title: "Member forced onto the relay path and restored to direct", Spec: []string{"§8", "AT31"}, Status: StatusFail}
	note := recPathNote(&rec)
	defer func() { r.record(rec) }()
	s := &Stack{lab: r.lab}
	defer func() { _, _ = s.ForceMemberRelay(context.Background(), "", false) }()
	if !r.flowCounted {
		rec.Status = StatusBlocked
		note("blocker", "counted member admission is the prerequisite")
		return
	}
	if !r.countedFlow(ctx, note) {
		note("owner", "member forwarding before relay fault")
		return
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.StopEstablishedTransports(c)
	}()
	before, err := s.ReadMemberPath(ctx, "")
	if err != nil {
		note("status_before", err.Error())
		return
	}
	note("relayed_before", itoa(before.Relayed))
	note("direct_before", itoa(before.Direct))
	forced, err := s.ForceMemberRelay(ctx, "", true)
	note("relayed_forced", itoa(forced.Relayed))
	note("direct_forced", itoa(forced.Direct))
	if err != nil {
		note("force", err.Error())
		note("owner", "w3-devstack-e2e: member was not observed on the relay path")
		return
	}
	if _, err = s.StartEstablishedTransports(ctx, r.usageDevice); err != nil {
		note("relay_traffic", err.Error())
		return
	}
	relayTraffic, _ := s.EstablishedStatus(ctx)
	note("relay_authenticated_udp", itoa(relayTraffic.UDPReplies))
	note("relay_tcp", itoa(relayTraffic.TCPEchoes))
	restored, err := s.ForceMemberRelay(ctx, "", false)
	note("relayed_restored", itoa(restored.Relayed))
	note("direct_restored", itoa(restored.Direct))
	if err != nil {
		note("restore", err.Error())
		note("owner", "w3-devstack-e2e: member did not return to the direct path")
		return
	}
	wait, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	continued := waitUntil(wait, time.Second, func() bool {
		status, e := s.EstablishedStatus(wait)
		return e == nil && status.TCPEchoes > relayTraffic.TCPEchoes && status.UDPReplies > relayTraffic.UDPReplies
	})
	note("same_sockets_continue_on_direct", boolText(continued))
	if !continued {
		return
	}
	note("table", "inet e2e_relay in the member; inet fvpn unchanged")
	rec.Status = StatusPass
}

func (r *runner) exitDNS(ctx context.Context) {
	rec := Record{ID: "exit-dns", Title: "Designated TCP and UDP DNS traverse the selected exit", Spec: []string{"§8.2", "§10.3", "AT12", "AT20"}, Status: StatusFail}
	note := recPathNote(&rec)
	defer func() { r.record(rec) }()
	if !r.flowCounted {
		rec.Status = StatusBlocked
		note("owner", "member admission prerequisite")
		return
	}
	s := &Stack{lab: r.lab}
	if !r.countedFlow(ctx, note) {
		note("owner", "member forwarding before DNS query")
		return
	}
	for _, tcp := range []bool{false, true} {
		observation, err := s.ObserveDesignatedDNS(ctx, r.usageDevice, tcp)
		raw, _ := json.Marshal(observation)
		note(observation.Transport, string(raw))
		if err != nil {
			note("error", err.Error())
			return
		}
		if observation.Reply != "rcode=0 answers=1" || observation.DNSUpload == 0 || observation.DNSDownload == 0 || observation.InternetUpload != 0 || observation.InternetDownload != 0 {
			note("owner", "w3-devstack-e2e / node enforcement: designated DNS path counters")
			return
		}
	}
	note("path", "origin-network DNS relay -> node filtering resolver; both nu_ and nd_ increased, u_ and d_ held")
	note("native_fallback", "not claimed; explicit resolver queries")
	rec.Status = StatusPass
}

func (r *runner) memberIPv6(ctx context.Context) {
	rec := Record{ID: "member-ipv6", Title: "Member IPv6 path", Spec: []string{"§8.3", "ISO03", "AT11"}, Status: StatusBlocked}
	note := recPathNote(&rec)
	defer func() { r.record(rec) }()
	addr, err := r.lab.docker(r.lab.member, "ip", "-6", "addr", "show", "dev", "fvnb0")
	text := strings.TrimSpace(string(addr))
	if err != nil {
		note("fvnb0_ipv6", "unavailable")
		text = ""
	} else {
		note("fvnb0_ipv6", clipText(text, 400))
	}
	raw, _ := r.lab.docker(r.lab.member, "sh", "-c", `sed -n 's/.*"DisableIPv6": *\(true\|false\).*/\1/p' /var/lib/netbird/default.json`)
	disabled := strings.TrimSpace(string(raw)) == "true"
	note("member_disable_ipv6", boolText(disabled))

	note("dns_resolver_ipv6", "empty in the prepared node config")
	note("blocker", "native member uses DisableIPv6; no IPv6 underlay or platform route/capture campaign is provisioned. An address or ctv6 rule alone cannot prove AT11.")
	note("owner", "w3-devstack-e2e / platform operators: dual-stack native path and per-platform leak measurements")
	_ = ctx
}
