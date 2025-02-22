package template

import (
	"regexp"
	"strconv"
	gotmpl "text/template"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/upstream"

	"github.com/horahoradev/dns"
)

func init() { plugin.Register("template", setupTemplate) }

func setupTemplate(c *caddy.Controller) error {
	handler, err := templateParse(c)
	if err != nil {
		return plugin.Error("template", err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		handler.Next = next
		return handler
	})

	return nil
}

func templateParse(c *caddy.Controller) (handler Handler, err error) {
	handler.Templates = make([]template, 0)

	for c.Next() {
		if !c.NextArg() {
			return handler, c.ArgErr()
		}
		class, ok := dns.StringToClass[c.Val()]
		if !ok {
			return handler, c.Errf("invalid query class %s", c.Val())
		}

		if !c.NextArg() {
			return handler, c.ArgErr()
		}
		qtype, ok := dns.StringToType[c.Val()]
		if !ok {
			return handler, c.Errf("invalid RR class %s", c.Val())
		}

		zones := plugin.OriginsFromArgsOrServerBlock(c.RemainingArgs(), c.ServerBlockKeys)
		handler.Zones = append(handler.Zones, zones...)
		t := template{qclass: class, qtype: qtype, zones: zones}

		t.regex = make([]*regexp.Regexp, 0)
		templatePrefix := ""

		t.answer = make([]*gotmpl.Template, 0)
		t.upstream = upstream.New()

		for c.NextBlock() {
			switch c.Val() {
			case "match":
				args := c.RemainingArgs()
				if len(args) == 0 {
					return handler, c.ArgErr()
				}
				for _, regex := range args {
					r, err := regexp.Compile(regex)
					if err != nil {
						return handler, c.Errf("could not parse regex: %s, %v", regex, err)
					}
					templatePrefix = templatePrefix + regex + " "
					t.regex = append(t.regex, r)
				}

			case "answer":
				args := c.RemainingArgs()
				if len(args) == 0 {
					return handler, c.ArgErr()
				}
				for _, answer := range args {
					tmpl, err := newTemplate("answer", answer)
					if err != nil {
						return handler, c.Errf("could not compile template: %s, %v", c.Val(), err)
					}
					t.answer = append(t.answer, tmpl)
				}

			case "additional":
				args := c.RemainingArgs()
				if len(args) == 0 {
					return handler, c.ArgErr()
				}
				for _, additional := range args {
					tmpl, err := newTemplate("additional", additional)
					if err != nil {
						return handler, c.Errf("could not compile template: %s, %v\n", c.Val(), err)
					}
					t.additional = append(t.additional, tmpl)
				}

			case "authority":
				args := c.RemainingArgs()
				if len(args) == 0 {
					return handler, c.ArgErr()
				}
				for _, authority := range args {
					tmpl, err := newTemplate("authority", authority)
					if err != nil {
						return handler, c.Errf("could not compile template: %s, %v\n", c.Val(), err)
					}
					t.authority = append(t.authority, tmpl)
				}

			case "rcode":
				if !c.NextArg() {
					return handler, c.ArgErr()
				}
				rcode, ok := dns.StringToRcode[c.Val()]
				if !ok {
					return handler, c.Errf("unknown rcode %s", c.Val())
				}
				t.rcode = rcode

			case "ederror":
				args := c.RemainingArgs()
				if len(args) != 1 && len(args) != 2 {
					return handler, c.ArgErr()
				}

				code, err := strconv.ParseUint(args[0], 10, 16)
				if err != nil {
					return handler, c.Errf("error parsing extended DNS error code %s, %v\n", c.Val(), err)
				}
				if len(args) == 2 {
					t.ederror = &ederror{code: uint16(code), reason: args[1]}
				} else {
					t.ederror = &ederror{code: uint16(code)}
				}

			case "fallthrough":
				t.fall.SetZonesFromArgs(c.RemainingArgs())

			case "upstream":
				// remove soon
				c.RemainingArgs()
			default:
				return handler, c.ArgErr()
			}
		}

		if len(t.regex) == 0 {
			t.regex = append(t.regex, regexp.MustCompile(".*"))
		}

		handler.Templates = append(handler.Templates, t)
	}

	return
}
