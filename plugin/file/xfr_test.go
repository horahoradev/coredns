package file

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"

	"github.com/horahoradev/dns"
)

func ExampleZone_All() {
	zone, err := Parse(strings.NewReader(dbMiekNL), testzone, "stdin", 0)
	if err != nil {
		return
	}
	records := zone.All()
	for _, r := range records {
		fmt.Printf("%+v\n", r)
	}
	// Output
	// xfr_test.go:15: miek.nl.	1800	IN	SOA	linode.atoom.net. miek.miek.nl. 1282630057 14400 3600 604800 14400
	// xfr_test.go:15: www.miek.nl.	1800	IN	CNAME	a.miek.nl.
	// xfr_test.go:15: miek.nl.	1800	IN	NS	linode.atoom.net.
	// xfr_test.go:15: miek.nl.	1800	IN	NS	ns-ext.nlnetlabs.nl.
	// xfr_test.go:15: miek.nl.	1800	IN	NS	omval.tednet.nl.
	// xfr_test.go:15: miek.nl.	1800	IN	NS	ext.ns.whyscream.net.
	// xfr_test.go:15: miek.nl.	1800	IN	MX	1 aspmx.l.google.com.
	// xfr_test.go:15: miek.nl.	1800	IN	MX	5 alt1.aspmx.l.google.com.
	// xfr_test.go:15: miek.nl.	1800	IN	MX	5 alt2.aspmx.l.google.com.
	// xfr_test.go:15: miek.nl.	1800	IN	MX	10 aspmx2.googlemail.com.
	// xfr_test.go:15: miek.nl.	1800	IN	MX	10 aspmx3.googlemail.com.
	// xfr_test.go:15: miek.nl.	1800	IN	A	139.162.196.78
	// xfr_test.go:15: miek.nl.	1800	IN	AAAA	2a01:7e00::f03c:91ff:fef1:6735
	// xfr_test.go:15: archive.miek.nl.	1800	IN	CNAME	a.miek.nl.
	// xfr_test.go:15: a.miek.nl.	1800	IN	A	139.162.196.78
	// xfr_test.go:15: a.miek.nl.	1800	IN	AAAA	2a01:7e00::f03c:91ff:fef1:6735
}

func TestAllNewZone(t *testing.T) {
	zone := NewZone("example.org.", "stdin")
	records := zone.All()
	if len(records) != 0 {
		t.Errorf("Expected %d records in empty zone, got %d", 0, len(records))
	}
}

func TestAXFRWithOutTransferPlugin(t *testing.T) {
	zone, err := Parse(strings.NewReader(dbMiekNL), testzone, "stdin", 0)
	if err != nil {
		t.Fatalf("Expected no error when reading zone, got %q", err)
	}

	fm := File{Next: test.ErrorHandler(), Zones: Zones{Z: map[string]*Zone{testzone: zone}, Names: []string{testzone}}}
	ctx := context.TODO()

	m := new(dns.Msg)
	m.SetQuestion("miek.nl.", dns.TypeAXFR)

	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	code, err := fm.ServeDNS(ctx, rec, m)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
		return
	}
	if code != dns.RcodeRefused {
		t.Errorf("Expecting REFUSED, got %d", code)
	}
}
