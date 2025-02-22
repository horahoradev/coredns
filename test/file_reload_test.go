package test

import (
	"os"
	"testing"
	"time"

	"github.com/coredns/coredns/plugin/test"

	"github.com/horahoradev/dns"
)

func TestZoneReload(t *testing.T) {
	name, rm, err := test.TempFile(".", exampleOrg)
	if err != nil {
		t.Fatalf("Failed to create zone: %s", err)
	}
	defer rm()

	// Corefile with two stanzas
	corefile := `
	example.org:0 {
		file ` + name + ` {
			reload 0.01s
		}
	}
	example.net:0 {
		file ` + name + `
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
		t.Fatalf("Expected to receive reply, but didn't: %s", err)
	}
	if len(resp.Answer) != 2 {
		t.Fatalf("Expected two RR in answer section got %d", len(resp.Answer))
	}

	// Remove RR from the Apex
	os.WriteFile(name, []byte(exampleOrgUpdated), 0644)

	time.Sleep(20 * time.Millisecond) // reload time, with some race insurance

	resp, err = dns.Exchange(m, udp)
	if err != nil {
		t.Fatal("Expected to receive reply, but didn't")
	}

	if len(resp.Answer) != 1 {
		t.Fatalf("Expected one RR in answer section got %d", len(resp.Answer))
	}
}

const exampleOrgUpdated = `; example.org test file
example.org.		IN	SOA	sns.dns.icann.org. noc.dns.icann.org. 2016082541 7200 3600 1209600 3600
example.org.		IN	NS	b.iana-servers.net.
example.org.		IN	NS	a.iana-servers.net.
example.org.		IN	A	127.0.0.2
`
