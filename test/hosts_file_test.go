package test

import (
	"testing"

	"github.com/horahoradev/dns"
)

func TestHostsInlineLookup(t *testing.T) {
	corefile := `example.org:0 {
		hosts highly_unlikely_to_exist_hosts_file example.org {
			10.0.0.1 example.org
			fallthrough
		}
	}`

	i, udp, _, err := CoreDNSServerAndPorts(corefile)
	if err != nil {
		t.Fatalf("Could not get CoreDNS serving instance: %s", err)
	}
	defer i.Stop()

	m := new(dns.Msg)
	m.SetQuestion("example.org.", dns.TypeA)
	resp, err := dns.Exchange(m, udp)
	if err != nil {
		t.Fatal("Expected to receive reply, but didn't")
	}
	// expect answer section with A record in it
	if len(resp.Answer) == 0 {
		t.Fatal("Expected to at least one RR in the answer section, got none")
	}
	if resp.Answer[0].Header().Rrtype != dns.TypeA {
		t.Errorf("Expected RR to A, got: %d", resp.Answer[0].Header().Rrtype)
	}
	if resp.Answer[0].(*dns.A).A.String() != "10.0.0.1" {
		t.Errorf("Expected 10.0.0.1, got: %s", resp.Answer[0].(*dns.A).A.String())
	}
}
