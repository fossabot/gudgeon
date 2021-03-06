package resolver

import (
	"testing"

	"github.com/miekg/dns"
)

func TestDnsSourceResolution(t *testing.T) {
	data := []struct {
		domain        string
		serverAddress string
		protocol      string
	}{
		// udp, regular port
		{"google.com.", "8.8.8.8", "udp"},
		{"cloudflare.com.", "1.1.1.1", "udp"},
		// tcp
		{"google.com.", "8.8.8.8", "tcp"},
		{"google.com.", "8.8.8.8/tcp", "tcp"},
		// tcp from udp regular port
		{"google.com.", "8.8.8.8/tcp", "udp"},
		// tcp-tls - disabled, not working consistently on travis-ci
		//{"google.com.", "8.8.8.8/tcp-tls", "tcp"},
		// udp, alternate ports
		{"google.com.", "208.67.222.222:5353", "udp"},
	}

	for _, d := range data {
		// create dns message from scratch
		m := &dns.Msg{
			MsgHdr: dns.MsgHdr{
				Authoritative:     true,
				AuthenticatedData: true,
				CheckingDisabled:  true,
				RecursionDesired:  true,
				Opcode:            dns.OpcodeQuery,
			},
			Question: make([]dns.Question, 1),
		}
		m.Question[0] = dns.Question{Name: d.domain, Qtype: dns.TypeA, Qclass: dns.ClassINET}

		// create source
		source := newDnsSource(d.serverAddress)

		// use source to resolve
		rCon := DefaultRequestContext()
		rCon.Protocol = d.protocol
		response, err := source.Answer(rCon, nil, m)
		if err != nil {
			t.Errorf("Could not resolve: %s", err)
			continue
		}

		// check response
		if len(response.Answer) < 1 {
			t.Errorf("No answers for question:\n%s\n-----\n%s", m, response)
			continue
		}
	}
}
