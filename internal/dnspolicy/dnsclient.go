package dnspolicy

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strings"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

// DNSAnswerRecord represents a parsed resource record from the Answer section
// of a DNS response.
type DNSAnswerRecord struct {
	Name     string          `json:"name"`
	Type     dnsmessage.Type `json:"type"`
	TTL      uint32          `json:"ttl"`
	IP       netip.Addr      `json:"ip,omitempty"`
	CNAME    string          `json:"cname,omitempty"`
	Target   string          `json:"target,omitempty"`
	Priority uint16          `json:"priority,omitempty"`
	TXT      string          `json:"txt,omitempty"`
}

// DNSResponse represents a decoded DNS response message.
type DNSResponse struct {
	ID           uint16            `json:"id"`
	RCode        dnsmessage.RCode  `json:"rcode"`
	Truncated    bool              `json:"truncated"`
	Answers      []DNSAnswerRecord `json:"answers"`
	AuthorityTTL uint32            `json:"authority_ttl,omitempty"`
	HasSOA       bool              `json:"has_soa"`
	Resolver     string            `json:"resolver"`
}

// IsNXDOMAIN reports whether the DNS response RCode is NXDOMAIN (RCodeNameError).
func (r *DNSResponse) IsNXDOMAIN() bool {
	return r != nil && r.RCode == dnsmessage.RCodeNameError
}

// IsSuccess reports whether the DNS response RCode is NOERROR (RCodeSuccess).
func (r *DNSResponse) IsSuccess() bool {
	return r != nil && r.RCode == dnsmessage.RCodeSuccess
}

// DNSClient sends DNS queries over UDP and TCP using golang.org/x/net/dns/dnsmessage.
type DNSClient struct {
	Timeout time.Duration
}

// NewDNSClient creates a DNSClient with the given per-query timeout (default 3s).
func NewDNSClient(timeout time.Duration) *DNSClient {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &DNSClient{Timeout: timeout}
}

// Query sends a DNS query for qname and qtype to serverAddr (host:port) over UDP,
// automatically retrying over TCP if the response has the TC (truncated) bit set.
func (c *DNSClient) Query(ctx context.Context, serverAddr, qname string, qtype dnsmessage.Type) (*DNSResponse, error) {
	resp, err := c.QueryProto(ctx, "udp", serverAddr, qname, qtype)
	if err != nil {
		return nil, err
	}
	if resp.Truncated {
		return c.QueryProto(ctx, "tcp", serverAddr, qname, qtype)
	}
	return resp, nil
}

// QueryProto sends a DNS query over the specified network ("udp" or "tcp").
func (c *DNSClient) QueryProto(ctx context.Context, network, serverAddr, qname string, qtype dnsmessage.Type) (*DNSResponse, error) {
	if strings.TrimSpace(serverAddr) == "" {
		return nil, errors.New("dns server address must not be empty")
	}
	fqdn := strings.TrimSpace(qname)
	if !strings.HasSuffix(fqdn, ".") {
		fqdn += "."
	}
	name, err := dnsmessage.NewName(fqdn)
	if err != nil {
		return nil, fmt.Errorf("invalid dns query name %q: %w", qname, err)
	}

	var idBytes [2]byte
	_, _ = rand.Read(idBytes[:])
	txID := binary.BigEndian.Uint16(idBytes[:])

	msg := dnsmessage.Message{
		Header: dnsmessage.Header{
			ID:               txID,
			RecursionDesired: true,
		},
		Questions: []dnsmessage.Question{
			{
				Name:  name,
				Type:  qtype,
				Class: dnsmessage.ClassINET,
			},
		},
	}
	wireQuery, err := msg.Pack()
	if err != nil {
		return nil, fmt.Errorf("pack dns query: %w", err)
	}

	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	queryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var dialer net.Dialer
	conn, err := dialer.DialContext(queryCtx, network, serverAddr)
	if err != nil {
		return nil, fmt.Errorf("dial dns %s %s: %w", network, serverAddr, err)
	}
	defer conn.Close()

	if deadline, ok := queryCtx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	var wireResp []byte
	switch network {
	case "tcp", "tcp4", "tcp6":
		lenBuf := make([]byte, 2+len(wireQuery))
		binary.BigEndian.PutUint16(lenBuf[:2], uint16(len(wireQuery)))
		copy(lenBuf[2:], wireQuery)
		if _, err := conn.Write(lenBuf); err != nil {
			return nil, fmt.Errorf("write tcp dns query: %w", err)
		}
		var respLenBuf [2]byte
		if _, err := io.ReadFull(conn, respLenBuf[:]); err != nil {
			return nil, fmt.Errorf("read tcp dns length: %w", err)
		}
		respLen := int(binary.BigEndian.Uint16(respLenBuf[:]))
		wireResp = make([]byte, respLen)
		if _, err := io.ReadFull(conn, wireResp); err != nil {
			return nil, fmt.Errorf("read tcp dns payload: %w", err)
		}
	default:
		if _, err := conn.Write(wireQuery); err != nil {
			return nil, fmt.Errorf("write udp dns query: %w", err)
		}
		buf := make([]byte, 4096)
		n, err := conn.Read(buf)
		if err != nil {
			return nil, fmt.Errorf("read udp dns response: %w", err)
		}
		wireResp = buf[:n]
	}

	return parseDNSWireResponse(wireResp, serverAddr)
}

func parseDNSWireResponse(wire []byte, serverAddr string) (*DNSResponse, error) {
	var p dnsmessage.Parser
	hdr, err := p.Start(wire)
	if err != nil {
		return nil, fmt.Errorf("parse dns header: %w", err)
	}
	if err := p.SkipAllQuestions(); err != nil {
		return nil, fmt.Errorf("skip dns questions: %w", err)
	}

	out := &DNSResponse{
		ID:        hdr.ID,
		RCode:     hdr.RCode,
		Truncated: hdr.Truncated,
		Resolver:  serverAddr,
	}

	for {
		ah, err := p.AnswerHeader()
		if errors.Is(err, dnsmessage.ErrSectionDone) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read answer header: %w", err)
		}
		rec := DNSAnswerRecord{
			Name: strings.TrimSuffix(ah.Name.String(), "."),
			Type: ah.Type,
			TTL:  ah.TTL,
		}
		switch ah.Type {
		case dnsmessage.TypeA:
			r, err := p.AResource()
			if err != nil {
				return nil, fmt.Errorf("read A resource: %w", err)
			}
			rec.IP = netip.AddrFrom4(r.A)
		case dnsmessage.TypeAAAA:
			r, err := p.AAAAResource()
			if err != nil {
				return nil, fmt.Errorf("read AAAA resource: %w", err)
			}
			rec.IP = netip.AddrFrom16(r.AAAA)
		case dnsmessage.TypeCNAME:
			r, err := p.CNAMEResource()
			if err != nil {
				return nil, fmt.Errorf("read CNAME resource: %w", err)
			}
			rec.CNAME = strings.TrimSuffix(r.CNAME.String(), ".")
		case dnsmessage.TypeHTTPS:
			r, err := p.HTTPSResource()
			if err != nil {
				return nil, fmt.Errorf("read HTTPS resource: %w", err)
			}
			rec.Priority = r.Priority
			rec.Target = strings.TrimSuffix(r.Target.String(), ".")
		case dnsmessage.TypeSVCB:
			r, err := p.SVCBResource()
			if err != nil {
				return nil, fmt.Errorf("read SVCB resource: %w", err)
			}
			rec.Priority = r.Priority
			rec.Target = strings.TrimSuffix(r.Target.String(), ".")
		case dnsmessage.TypeTXT:
			r, err := p.TXTResource()
			if err != nil {
				return nil, fmt.Errorf("read TXT resource: %w", err)
			}
			rec.TXT = strings.Join(r.TXT, "")
		default:
			if err := p.SkipAnswer(); err != nil {
				return nil, fmt.Errorf("skip answer: %w", err)
			}
		}
		out.Answers = append(out.Answers, rec)
	}

	for {
		authHdr, err := p.AuthorityHeader()
		if errors.Is(err, dnsmessage.ErrSectionDone) {
			break
		}
		if err != nil {
			break
		}
		if authHdr.Type == dnsmessage.TypeSOA {
			out.HasSOA = true
			out.AuthorityTTL = authHdr.TTL
			_, _ = p.SOAResource()
		} else {
			_ = p.SkipAuthority()
		}
	}

	return out, nil
}

// QueryWithFailover queries the configured controlled resolvers in order. If one
// resolver is down, the next controlled resolver answers. If all configured
// resolvers are unreachable, QueryWithFailover fails closed and never falls back
// to any unconfigured public or host DNS resolver (§12.2, §12.5, §17.1, AT21).
func (c *DNSClient) QueryWithFailover(ctx context.Context, resolvers []string, qname string, qtype dnsmessage.Type) (*DNSResponse, error) {
	if len(resolvers) == 0 {
		return nil, errors.New("no controlled DNS resolvers configured (fail-closed)")
	}
	var errs []error
	for _, addr := range resolvers {
		resp, err := c.Query(ctx, addr, qname, qtype)
		if err == nil {
			return resp, nil
		}
		errs = append(errs, fmt.Errorf("resolver %s: %w", addr, err))
	}
	return nil, fmt.Errorf("all %d controlled DNS resolvers failed (no unfiltered fallback): %w", len(resolvers), errors.Join(errs...))
}
