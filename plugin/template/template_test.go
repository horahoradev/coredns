package template

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	gotmpl "text/template"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/metadata"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/plugin/test"

	"github.com/horahoradev/dns"
)

func TestHandler(t *testing.T) {
	exampleDomainATemplate := template{
		regex:  []*regexp.Regexp{regexp.MustCompile("(^|[.])ip-10-(?P<b>[0-9]*)-(?P<c>[0-9]*)-(?P<d>[0-9]*)[.]example[.]$")},
		answer: []*gotmpl.Template{gotmpl.Must(newTemplate("answer", "{{ .Name }} 60 IN A 10.{{ .Group.b }}.{{ .Group.c }}.{{ .Group.d }}"))},
		qclass: dns.ClassANY,
		qtype:  dns.TypeANY,
		fall:   fall.Root,
		zones:  []string{"."},
	}
	exampleDomainAParseIntTemplate := template{
		regex:  []*regexp.Regexp{regexp.MustCompile("^ip0a(?P<b>[a-f0-9]{2})(?P<c>[a-f0-9]{2})(?P<d>[a-f0-9]{2})[.]example[.]$")},
		answer: []*gotmpl.Template{gotmpl.Must(newTemplate("answer", "{{ .Name }} 60 IN A 10.{{ parseInt .Group.b 16 8 }}.{{ parseInt .Group.c 16 8 }}.{{ parseInt .Group.d 16 8 }}"))},
		qclass: dns.ClassANY,
		qtype:  dns.TypeANY,
		fall:   fall.Root,
		zones:  []string{"."},
	}
	exampleDomainIPATemplate := template{
		regex:  []*regexp.Regexp{regexp.MustCompile(".*")},
		answer: []*gotmpl.Template{gotmpl.Must(newTemplate("answer", "{{ .Name }} 60 IN A {{ .Remote }}"))},
		qclass: dns.ClassINET,
		qtype:  dns.TypeA,
		fall:   fall.Root,
		zones:  []string{"."},
	}
	exampleDomainANSTemplate := template{
		regex:      []*regexp.Regexp{regexp.MustCompile("(^|[.])ip-10-(?P<b>[0-9]*)-(?P<c>[0-9]*)-(?P<d>[0-9]*)[.]example[.]$")},
		answer:     []*gotmpl.Template{gotmpl.Must(newTemplate("answer", "{{ .Name }} 60 IN A 10.{{ .Group.b }}.{{ .Group.c }}.{{ .Group.d }}"))},
		additional: []*gotmpl.Template{gotmpl.Must(newTemplate("additional", "ns0.example. IN A 203.0.113.8"))},
		authority:  []*gotmpl.Template{gotmpl.Must(newTemplate("authority", "example. IN NS ns0.example.com."))},
		qclass:     dns.ClassANY,
		qtype:      dns.TypeANY,
		fall:       fall.Root,
		zones:      []string{"."},
	}
	exampleDomainMXTemplate := template{
		regex:      []*regexp.Regexp{regexp.MustCompile("(^|[.])ip-10-(?P<b>[0-9]*)-(?P<c>[0-9]*)-(?P<d>[0-9]*)[.]example[.]$")},
		answer:     []*gotmpl.Template{gotmpl.Must(newTemplate("answer", "{{ .Name }} 60 MX 10 {{ .Name }}"))},
		additional: []*gotmpl.Template{gotmpl.Must(newTemplate("additional", "{{ .Name }} 60 IN A 10.{{ .Group.b }}.{{ .Group.c }}.{{ .Group.d }}"))},
		qclass:     dns.ClassANY,
		qtype:      dns.TypeANY,
		fall:       fall.Root,
		zones:      []string{"."},
	}
	invalidDomainTemplate := template{
		regex:  []*regexp.Regexp{regexp.MustCompile("[.]invalid[.]$")},
		rcode:  dns.RcodeNameError,
		answer: []*gotmpl.Template{gotmpl.Must(newTemplate("answer", "invalid. 60 {{ .Class }} SOA a.invalid. b.invalid. (1 60 60 60 60)"))},
		qclass: dns.ClassANY,
		qtype:  dns.TypeANY,
		fall:   fall.Root,
		zones:  []string{"."},
	}
	rcodeServfailTemplate := template{
		regex:  []*regexp.Regexp{regexp.MustCompile(".*")},
		rcode:  dns.RcodeServerFailure,
		qclass: dns.ClassANY,
		qtype:  dns.TypeANY,
		fall:   fall.Root,
		zones:  []string{"."},
	}
	brokenTemplate := template{
		regex:  []*regexp.Regexp{regexp.MustCompile("[.]example[.]$")},
		answer: []*gotmpl.Template{gotmpl.Must(newTemplate("answer", "{{ .Name }} 60 IN TXT \"{{ index .Match 2 }}\""))},
		qclass: dns.ClassANY,
		qtype:  dns.TypeANY,
		fall:   fall.Root,
		zones:  []string{"."},
	}
	brokenParseIntTemplate := template{
		regex:  []*regexp.Regexp{regexp.MustCompile("[.]example[.]$")},
		answer: []*gotmpl.Template{gotmpl.Must(newTemplate("answer", "{{ .Name }} 60 IN TXT \"{{ parseInt \"gg\" 16 8 }}\""))},
		qclass: dns.ClassANY,
		qtype:  dns.TypeANY,
		fall:   fall.Root,
		zones:  []string{"."},
	}
	nonRRTemplate := template{
		regex:  []*regexp.Regexp{regexp.MustCompile("[.]example[.]$")},
		answer: []*gotmpl.Template{gotmpl.Must(newTemplate("answer", "{{ .Name }}"))},
		qclass: dns.ClassANY,
		qtype:  dns.TypeANY,
		fall:   fall.Root,
		zones:  []string{"."},
	}
	nonRRAdditionalTemplate := template{
		regex:      []*regexp.Regexp{regexp.MustCompile("[.]example[.]$")},
		additional: []*gotmpl.Template{gotmpl.Must(newTemplate("answer", "{{ .Name }}"))},
		qclass:     dns.ClassANY,
		qtype:      dns.TypeANY,
		fall:       fall.Root,
		zones:      []string{"."},
	}
	nonRRAuthoritativeTemplate := template{
		regex:     []*regexp.Regexp{regexp.MustCompile("[.]example[.]$")},
		authority: []*gotmpl.Template{gotmpl.Must(newTemplate("answer", "{{ .Name }}"))},
		qclass:    dns.ClassANY,
		qtype:     dns.TypeANY,
		fall:      fall.Root,
		zones:     []string{"."},
	}
	cnameTemplate := template{
		regex:  []*regexp.Regexp{regexp.MustCompile("example[.]net[.]")},
		answer: []*gotmpl.Template{gotmpl.Must(newTemplate("answer", "example.net 60 IN CNAME target.example.com"))},
		qclass: dns.ClassANY,
		qtype:  dns.TypeANY,
		fall:   fall.Root,
		zones:  []string{"."},
	}
	mdTemplate := template{
		regex:      []*regexp.Regexp{regexp.MustCompile("(^|[.])ip-10-(?P<b>[0-9]*)-(?P<c>[0-9]*)-(?P<d>[0-9]*)[.]example[.]$")},
		answer:     []*gotmpl.Template{gotmpl.Must(newTemplate("answer", `{{ .Meta "foo" }}-{{ .Name }} 60 IN A 10.{{ .Group.b }}.{{ .Group.c }}.{{ .Group.d }}`))},
		additional: []*gotmpl.Template{gotmpl.Must(newTemplate("additional", `{{ .Meta "bar" }}.example. IN A 203.0.113.8`))},
		authority:  []*gotmpl.Template{gotmpl.Must(newTemplate("authority", `example. IN NS {{ .Meta "bar" }}.example.com.`))},
		qclass:     dns.ClassANY,
		qtype:      dns.TypeANY,
		fall:       fall.Root,
		zones:      []string{"."},
	}
	mdMissingTemplate := template{
		regex:  []*regexp.Regexp{regexp.MustCompile("(^|[.])ip-10-(?P<b>[0-9]*)-(?P<c>[0-9]*)-(?P<d>[0-9]*)[.]example[.]$")},
		answer: []*gotmpl.Template{gotmpl.Must(newTemplate("answer", `{{ .Meta "foofoo" }}{{ .Name }} 60 IN A 10.{{ .Group.b }}.{{ .Group.c }}.{{ .Group.d }}`))},
		qclass: dns.ClassANY,
		qtype:  dns.TypeANY,
		fall:   fall.Root,
		zones:  []string{"."},
	}
	templateWithEDE := template{
		rcode:     dns.RcodeNameError,
		regex:     []*regexp.Regexp{regexp.MustCompile(".*")},
		authority: []*gotmpl.Template{gotmpl.Must(newTemplate("authority", "invalid. 60 {{ .Class }} SOA ns.invalid. hostmaster.invalid. (1 60 60 60 60)"))},
		qclass:    dns.ClassANY,
		qtype:     dns.TypeANY,
		fall:      fall.Root,
		zones:     []string{"."},
		ederror:   &ederror{code: 21, reason: "Blocked due to RFC2606"},
	}

	tests := []struct {
		tmpl           template
		qname          string
		name           string
		qclass         uint16
		qtype          uint16
		expectedCode   int
		expectedErr    string
		verifyResponse func(*dns.Msg) error
		md             map[string]string
	}{
		{
			name:         "RcodeServFail",
			tmpl:         rcodeServfailTemplate,
			qname:        "test.invalid.",
			expectedCode: dns.RcodeServerFailure,
			verifyResponse: func(r *dns.Msg) error {
				return nil
			},
		},
		{
			name:         "ExampleDomainNameMismatch",
			tmpl:         exampleDomainATemplate,
			qclass:       dns.ClassINET,
			qtype:        dns.TypeA,
			qname:        "test.invalid.",
			expectedCode: rcodeFallthrough,
		},
		{
			name:         "BrokenTemplate",
			tmpl:         brokenTemplate,
			qclass:       dns.ClassINET,
			qtype:        dns.TypeANY,
			qname:        "test.example.",
			expectedCode: dns.RcodeServerFailure,
			expectedErr:  `template: answer:1:26: executing "answer" at <index .Match 2>: error calling index: index out of range: 2`,
			verifyResponse: func(r *dns.Msg) error {
				return nil
			},
		},
		{
			name:         "NonRRTemplate",
			tmpl:         nonRRTemplate,
			qclass:       dns.ClassINET,
			qtype:        dns.TypeANY,
			qname:        "test.example.",
			expectedCode: dns.RcodeServerFailure,
			expectedErr:  `dns: not a TTL: "test.example." at line: 1:13`,
			verifyResponse: func(r *dns.Msg) error {
				return nil
			},
		},
		{
			name:         "NonRRAdditionalTemplate",
			tmpl:         nonRRAdditionalTemplate,
			qclass:       dns.ClassINET,
			qtype:        dns.TypeANY,
			qname:        "test.example.",
			expectedCode: dns.RcodeServerFailure,
			expectedErr:  `dns: not a TTL: "test.example." at line: 1:13`,
			verifyResponse: func(r *dns.Msg) error {
				return nil
			},
		},
		{
			name:         "NonRRAuthorityTemplate",
			tmpl:         nonRRAuthoritativeTemplate,
			qclass:       dns.ClassINET,
			qtype:        dns.TypeANY,
			qname:        "test.example.",
			expectedCode: dns.RcodeServerFailure,
			expectedErr:  `dns: not a TTL: "test.example." at line: 1:13`,
			verifyResponse: func(r *dns.Msg) error {
				return nil
			},
		},
		{
			name:   "ExampleIPMatch",
			tmpl:   exampleDomainIPATemplate,
			qclass: dns.ClassINET,
			qtype:  dns.TypeA,
			qname:  "test.example.",
			verifyResponse: func(r *dns.Msg) error {
				if len(r.Answer) != 1 {
					return fmt.Errorf("expected 1 answer, got %v", len(r.Answer))
				}
				if r.Answer[0].Header().Rrtype != dns.TypeA {
					return fmt.Errorf("expected an A record answer, got %v", dns.TypeToString[r.Answer[0].Header().Rrtype])
				}
				if r.Answer[0].(*dns.A).A.String() != "10.240.0.1" {
					return fmt.Errorf("expected an A record for 10.95.12.8, got %v", r.Answer[0].String())
				}
				return nil
			},
		},
		{
			name:   "ExampleDomainMatch",
			tmpl:   exampleDomainATemplate,
			qclass: dns.ClassINET,
			qtype:  dns.TypeA,
			qname:  "ip-10-95-12-8.example.",
			verifyResponse: func(r *dns.Msg) error {
				if len(r.Answer) != 1 {
					return fmt.Errorf("expected 1 answer, got %v", len(r.Answer))
				}
				if r.Answer[0].Header().Rrtype != dns.TypeA {
					return fmt.Errorf("expected an A record answer, got %v", dns.TypeToString[r.Answer[0].Header().Rrtype])
				}
				if r.Answer[0].(*dns.A).A.String() != "10.95.12.8" {
					return fmt.Errorf("expected an A record for 10.95.12.8, got %v", r.Answer[0].String())
				}
				return nil
			},
		},
		{
			name:   "ExampleDomainMatchHexIp",
			tmpl:   exampleDomainAParseIntTemplate,
			qclass: dns.ClassINET,
			qtype:  dns.TypeA,
			qname:  "ip0a5f0c09.example.",
			verifyResponse: func(r *dns.Msg) error {
				if len(r.Answer) != 1 {
					return fmt.Errorf("expected 1 answer, got %v", len(r.Answer))
				}
				if r.Answer[0].Header().Rrtype != dns.TypeA {
					return fmt.Errorf("expected an A record answer, got %v", dns.TypeToString[r.Answer[0].Header().Rrtype])
				}
				if r.Answer[0].(*dns.A).A.String() != "10.95.12.9" {
					return fmt.Errorf("expected an A record for 10.95.12.9, got %v", r.Answer[0].String())
				}
				return nil
			},
		},
		{
			name:         "BrokenParseIntTemplate",
			tmpl:         brokenParseIntTemplate,
			qclass:       dns.ClassINET,
			qtype:        dns.TypeANY,
			qname:        "test.example.",
			expectedCode: dns.RcodeServerFailure,
			expectedErr:  "template: answer:1:26: executing \"answer\" at <parseInt \"gg\" 16 8>: error calling parseInt: strconv.ParseUint: parsing \"gg\": invalid syntax",
			verifyResponse: func(r *dns.Msg) error {
				return nil
			},
		},
		{
			name:   "ExampleDomainMXMatch",
			tmpl:   exampleDomainMXTemplate,
			qclass: dns.ClassINET,
			qtype:  dns.TypeMX,
			qname:  "ip-10-95-12-8.example.",
			verifyResponse: func(r *dns.Msg) error {
				if len(r.Answer) != 1 {
					return fmt.Errorf("expected 1 answer, got %v", len(r.Answer))
				}
				if r.Answer[0].Header().Rrtype != dns.TypeMX {
					return fmt.Errorf("expected an A record answer, got %v", dns.TypeToString[r.Answer[0].Header().Rrtype])
				}
				if len(r.Extra) != 1 {
					return fmt.Errorf("expected 1 extra record, got %v", len(r.Extra))
				}
				if r.Extra[0].Header().Rrtype != dns.TypeA {
					return fmt.Errorf("expected an additional A record, got %v", dns.TypeToString[r.Extra[0].Header().Rrtype])
				}
				return nil
			},
		},
		{
			name:   "ExampleDomainANSMatch",
			tmpl:   exampleDomainANSTemplate,
			qclass: dns.ClassINET,
			qtype:  dns.TypeA,
			qname:  "ip-10-95-12-8.example.",
			verifyResponse: func(r *dns.Msg) error {
				if len(r.Answer) != 1 {
					return fmt.Errorf("expected 1 answer, got %v", len(r.Answer))
				}
				if r.Answer[0].Header().Rrtype != dns.TypeA {
					return fmt.Errorf("expected an A record answer, got %v", dns.TypeToString[r.Answer[0].Header().Rrtype])
				}
				if len(r.Extra) != 1 {
					return fmt.Errorf("expected 1 extra record, got %v", len(r.Extra))
				}
				if r.Extra[0].Header().Rrtype != dns.TypeA {
					return fmt.Errorf("expected an additional A record, got %v", dns.TypeToString[r.Extra[0].Header().Rrtype])
				}
				if len(r.Ns) != 1 {
					return fmt.Errorf("expected 1 authoritative record, got %v", len(r.Extra))
				}
				if r.Ns[0].Header().Rrtype != dns.TypeNS {
					return fmt.Errorf("expected an authoritative NS record, got %v", dns.TypeToString[r.Extra[0].Header().Rrtype])
				}
				return nil
			},
		},
		{
			name:         "ExampleInvalidNXDOMAIN",
			tmpl:         invalidDomainTemplate,
			qclass:       dns.ClassINET,
			qtype:        dns.TypeMX,
			qname:        "test.invalid.",
			expectedCode: dns.RcodeNameError,
			verifyResponse: func(r *dns.Msg) error {
				if len(r.Answer) != 1 {
					return fmt.Errorf("expected 1 answer, got %v", len(r.Answer))
				}
				if r.Answer[0].Header().Rrtype != dns.TypeSOA {
					return fmt.Errorf("expected an SOA record answer, got %v", dns.TypeToString[r.Answer[0].Header().Rrtype])
				}
				return nil
			},
		},
		{
			name:         "CNAMEWithoutUpstream",
			tmpl:         cnameTemplate,
			qclass:       dns.ClassINET,
			qtype:        dns.TypeA,
			qname:        "example.net.",
			expectedCode: dns.RcodeSuccess,
			verifyResponse: func(r *dns.Msg) error {
				if len(r.Answer) != 1 {
					return fmt.Errorf("expected 1 answer, got %v", len(r.Answer))
				}
				return nil
			},
		},
		{
			name:   "mdMatch",
			tmpl:   mdTemplate,
			qclass: dns.ClassINET,
			qtype:  dns.TypeA,
			qname:  "ip-10-95-12-8.example.",
			verifyResponse: func(r *dns.Msg) error {
				if len(r.Answer) != 1 {
					return fmt.Errorf("expected 1 answer, got %v", len(r.Answer))
				}
				if r.Answer[0].Header().Rrtype != dns.TypeA {
					return fmt.Errorf("expected an A record answer, got %v", dns.TypeToString[r.Answer[0].Header().Rrtype])
				}
				name := "myfoo-ip-10-95-12-8.example."
				if r.Answer[0].Header().Name != name {
					return fmt.Errorf("expected answer name %q, got %q", name, r.Answer[0].Header().Name)
				}
				if len(r.Extra) != 1 {
					return fmt.Errorf("expected 1 extra record, got %v", len(r.Extra))
				}
				if r.Extra[0].Header().Rrtype != dns.TypeA {
					return fmt.Errorf("expected an additional A record, got %v", dns.TypeToString[r.Extra[0].Header().Rrtype])
				}
				name = "mybar.example."
				if r.Extra[0].Header().Name != name {
					return fmt.Errorf("expected additional name %q, got %q", name, r.Extra[0].Header().Name)
				}
				if len(r.Ns) != 1 {
					return fmt.Errorf("expected 1 authoritative record, got %v", len(r.Extra))
				}
				if r.Ns[0].Header().Rrtype != dns.TypeNS {
					return fmt.Errorf("expected an authoritative NS record, got %v", dns.TypeToString[r.Extra[0].Header().Rrtype])
				}
				ns, ok := r.Ns[0].(*dns.NS)
				if !ok {
					return fmt.Errorf("expected NS record to be type NS, got %v", r.Ns[0])
				}
				rdata := "mybar.example.com."
				if ns.Ns != rdata {
					return fmt.Errorf("expected ns rdata %q, got %q", rdata, ns.Ns)
				}
				return nil
			},
			md: map[string]string{
				"foo":    "myfoo",
				"bar":    "mybar",
				"foobar": "myfoobar",
			},
		},
		{
			name:   "mdMissing",
			tmpl:   mdMissingTemplate,
			qclass: dns.ClassINET,
			qtype:  dns.TypeA,
			qname:  "ip-10-95-12-8.example.",
			verifyResponse: func(r *dns.Msg) error {
				if len(r.Answer) != 1 {
					return fmt.Errorf("expected 1 answer, got %v", len(r.Answer))
				}
				if r.Answer[0].Header().Rrtype != dns.TypeA {
					return fmt.Errorf("expected an A record answer, got %v", dns.TypeToString[r.Answer[0].Header().Rrtype])
				}
				name := "ip-10-95-12-8.example."
				if r.Answer[0].Header().Name != name {
					return fmt.Errorf("expected answer name %q, got %q", name, r.Answer[0].Header().Name)
				}
				return nil
			},
			md: map[string]string{
				"foo": "myfoo",
			},
		},
		{
			name:         "EDNS error",
			tmpl:         templateWithEDE,
			qclass:       dns.ClassINET,
			qtype:        dns.TypeA,
			qname:        "test.invalid.",
			expectedCode: dns.RcodeNameError,
			verifyResponse: func(r *dns.Msg) error {
				if opt := r.IsEdns0(); opt != nil {
					matched := false
					for _, ednsopt := range opt.Option {
						if ede, ok := ednsopt.(*dns.EDNS0_EDE); ok {
							if ede.InfoCode != dns.ExtendedErrorCodeNotSupported {
								return fmt.Errorf("unexpected EDE code = %v, want %v", ede.InfoCode, dns.ExtendedErrorCodeNotSupported)
							}
							matched = true
						}
					}
					if !matched {
						t.Error("Error: acl.ServeDNS() missing Extended DNS Error option")
					}
				} else {
					return fmt.Errorf("expected EDNS enabled")
				}
				return nil
			},
		},
	}

	ctx := context.TODO()

	for _, tr := range tests {
		handler := Handler{
			Next:      test.NextHandler(rcodeFallthrough, nil),
			Zones:     []string{"."},
			Templates: []template{tr.tmpl},
		}
		req := &dns.Msg{
			Question: []dns.Question{{
				Name:   tr.qname,
				Qclass: tr.qclass,
				Qtype:  tr.qtype,
			}},
		}
		rec := dnstest.NewRecorder(&test.ResponseWriter{})
		if tr.md != nil {
			ctx = metadata.ContextWithMetadata(context.Background())

			for k, v := range tr.md {
				// Go requires copying to a local variable for the closure to work
				kk := k
				vv := v
				metadata.SetValueFunc(ctx, kk, func() string {
					return vv
				})
			}
		}

		code, err := handler.ServeDNS(ctx, rec, req)
		if err == nil && tr.expectedErr != "" {
			t.Errorf("Test %v expected error: %v, got nothing", tr.name, tr.expectedErr)
		}
		if err != nil && tr.expectedErr == "" {
			t.Errorf("Test %v expected no error got: %v", tr.name, err)
		}
		if err != nil && tr.expectedErr != "" && err.Error() != tr.expectedErr {
			t.Errorf("Test %v expected error: %v, got: %v", tr.name, tr.expectedErr, err)
		}
		if code != tr.expectedCode {
			t.Errorf("Test %v expected response code %v, got %v", tr.name, tr.expectedCode, code)
		}
		if err == nil && code != rcodeFallthrough {
			// only verify if we got no error and expected no error
			if err := tr.verifyResponse(rec.Msg); err != nil {
				t.Errorf("Test %v could not verify the response: %v", tr.name, err)
			}
		}
	}
}

// TestMultiSection verifies that a corefile with multiple but different template sections works
func TestMultiSection(t *testing.T) {
	ctx := context.TODO()

	multisectionConfig := `
		# Implicit section (see c.ServerBlockKeys)
		# test.:8053 {

	  # REFUSE IN A for the server zone (test.)
		template IN A {
			rcode REFUSED
		}
		# Fallthrough everything IN TXT for test.
		template IN TXT {
			match "$^"
			rcode SERVFAIL
			fallthrough
		}
		# Answer CH TXT *.coredns.invalid. / coredns.invalid.
		template CH TXT coredns.invalid {
			answer "{{ .Name }} 60 CH TXT \"test\""
		}
		# Answer example. ip templates and fallthrough otherwise
		template IN A example {
			match ^ip-10-(?P<b>[0-9]*)-(?P<c>[0-9]*)-(?P<d>[0-9]*)[.]example[.]$
			answer "{{ .Name }} 60 IN A 10.{{ .Group.b }}.{{ .Group.c }}.{{ .Group.d }}"
			fallthrough
		}
		# Answer MX record requests for ip templates in example. and never fall through
		template IN MX example {
			match ^ip-10-(?P<b>[0-9]*)-(?P<c>[0-9]*)-(?P<d>[0-9]*)[.]example[.]$
			answer "{{ .Name }} 60 IN MX 10 {{ .Name }}"
			additional "{{ .Name }} 60 IN A 10.{{ .Group.b }}.{{ .Group.c }}.{{ .Group.d }}"
		}
		`
	c := caddy.NewTestController("dns", multisectionConfig)
	c.ServerBlockKeys = []string{"test.:8053"}

	handler, err := templateParse(c)
	if err != nil {
		t.Fatalf("TestMultiSection could not parse config: %v", err)
	}

	handler.Next = test.NextHandler(rcodeFallthrough, nil)

	rec := dnstest.NewRecorder(&test.ResponseWriter{})

	// Asking for test. IN A -> REFUSED

	req := &dns.Msg{Question: []dns.Question{{Name: "some.test.", Qclass: dns.ClassINET, Qtype: dns.TypeA}}}
	code, err := handler.ServeDNS(ctx, rec, req)
	if err != nil {
		t.Fatalf("TestMultiSection expected no error resolving some.test. A, got: %v", err)
	}
	if code != dns.RcodeRefused {
		t.Fatalf("TestMultiSection expected response code REFUSED got: %v", code)
	}

	// Asking for test. IN TXT -> fallthrough

	req = &dns.Msg{Question: []dns.Question{{Name: "some.test.", Qclass: dns.ClassINET, Qtype: dns.TypeTXT}}}
	code, err = handler.ServeDNS(ctx, rec, req)
	if err != nil {
		t.Fatalf("TestMultiSection expected no error resolving some.test. TXT, got: %v", err)
	}
	if code != rcodeFallthrough {
		t.Fatalf("TestMultiSection expected response code fallthrough got: %v", code)
	}

	// Asking for coredns.invalid. CH TXT -> TXT "test"

	req = &dns.Msg{Question: []dns.Question{{Name: "coredns.invalid.", Qclass: dns.ClassCHAOS, Qtype: dns.TypeTXT}}}
	code, err = handler.ServeDNS(ctx, rec, req)
	if err != nil {
		t.Fatalf("TestMultiSection expected no error resolving coredns.invalid. TXT, got: %v", err)
	}
	if code != dns.RcodeSuccess {
		t.Fatalf("TestMultiSection expected success response for coredns.invalid. TXT got: %v", code)
	}
	if len(rec.Msg.Answer) != 1 {
		t.Fatalf("TestMultiSection expected one answer for coredns.invalid. TXT got: %v", rec.Msg.Answer)
	}
	if rec.Msg.Answer[0].Header().Rrtype != dns.TypeTXT || rec.Msg.Answer[0].(*dns.TXT).Txt[0] != "test" {
		t.Fatalf("TestMultiSection a \"test\" answer for coredns.invalid. TXT got: %v", rec.Msg.Answer[0])
	}

	// Asking for an ip template in example

	req = &dns.Msg{Question: []dns.Question{{Name: "ip-10-11-12-13.example.", Qclass: dns.ClassINET, Qtype: dns.TypeA}}}
	code, err = handler.ServeDNS(ctx, rec, req)
	if err != nil {
		t.Fatalf("TestMultiSection expected no error resolving ip-10-11-12-13.example. IN A, got: %v", err)
	}
	if code != dns.RcodeSuccess {
		t.Fatalf("TestMultiSection expected success response ip-10-11-12-13.example. IN A got: %v, %v", code, dns.RcodeToString[code])
	}
	if len(rec.Msg.Answer) != 1 {
		t.Fatalf("TestMultiSection expected one answer for ip-10-11-12-13.example. IN A got: %v", rec.Msg.Answer)
	}
	if rec.Msg.Answer[0].Header().Rrtype != dns.TypeA {
		t.Fatalf("TestMultiSection an A RR answer for ip-10-11-12-13.example. IN A got: %v", rec.Msg.Answer[0])
	}

	// Asking for an MX ip template in example

	req = &dns.Msg{Question: []dns.Question{{Name: "ip-10-11-12-13.example.", Qclass: dns.ClassINET, Qtype: dns.TypeMX}}}
	code, err = handler.ServeDNS(ctx, rec, req)
	if err != nil {
		t.Fatalf("TestMultiSection expected no error resolving ip-10-11-12-13.example. IN MX, got: %v", err)
	}
	if code != dns.RcodeSuccess {
		t.Fatalf("TestMultiSection expected success response ip-10-11-12-13.example. IN MX got: %v, %v", code, dns.RcodeToString[code])
	}
	if len(rec.Msg.Answer) != 1 {
		t.Fatalf("TestMultiSection expected one answer for ip-10-11-12-13.example. IN MX got: %v", rec.Msg.Answer)
	}
	if rec.Msg.Answer[0].Header().Rrtype != dns.TypeMX {
		t.Fatalf("TestMultiSection an A RR answer for ip-10-11-12-13.example. IN MX got: %v", rec.Msg.Answer[0])
	}

	// Test that something.example. A does fall through but something.example. MX does not

	req = &dns.Msg{Question: []dns.Question{{Name: "something.example.", Qclass: dns.ClassINET, Qtype: dns.TypeA}}}
	code, err = handler.ServeDNS(ctx, rec, req)
	if err != nil {
		t.Fatalf("TestMultiSection expected no error resolving something.example. IN A, got: %v", err)
	}
	if code != rcodeFallthrough {
		t.Fatalf("TestMultiSection expected a fall through resolving something.example. IN A, got: %v, %v", code, dns.RcodeToString[code])
	}

	req = &dns.Msg{Question: []dns.Question{{Name: "something.example.", Qclass: dns.ClassINET, Qtype: dns.TypeMX}}}
	code, err = handler.ServeDNS(ctx, rec, req)
	if err != nil {
		t.Fatalf("TestMultiSection expected no error resolving something.example. IN MX, got: %v", err)
	}
	if code == rcodeFallthrough {
		t.Fatalf("TestMultiSection expected no fall through resolving something.example. IN MX")
	}
	if code != dns.RcodeServerFailure {
		t.Fatalf("TestMultiSection expected SERVFAIL resolving something.example. IN MX, got %v, %v", code, dns.RcodeToString[code])
	}
}

const rcodeFallthrough = 3841 // reserved for private use, used to indicate a fallthrough
