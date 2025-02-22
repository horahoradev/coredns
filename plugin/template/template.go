package template

import (
	"bytes"
	"context"
	"regexp"
	"strconv"
	gotmpl "text/template"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metadata"
	"github.com/coredns/coredns/plugin/metrics"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/request"

	"github.com/horahoradev/dns"
)

// Handler is a plugin handler that takes a query and templates a response.
type Handler struct {
	Zones []string

	Next      plugin.Handler
	Templates []template
}

type template struct {
	zones      []string
	rcode      int
	regex      []*regexp.Regexp
	answer     []*gotmpl.Template
	additional []*gotmpl.Template
	authority  []*gotmpl.Template
	qclass     uint16
	qtype      uint16
	ederror    *ederror
	fall       fall.F
	upstream   Upstreamer
}

type ederror struct {
	code   uint16
	reason string
}

// Upstreamer looks up targets of CNAME templates
type Upstreamer interface {
	Lookup(ctx context.Context, state request.Request, name string, typ uint16) (*dns.Msg, error)
}

type templateData struct {
	Zone     string
	Name     string
	Regex    string
	Match    []string
	Group    map[string]string
	Class    string
	Type     string
	Message  *dns.Msg
	Question *dns.Question
	Remote   string
	md       map[string]metadata.Func
}

func (data *templateData) Meta(metaName string) string {
	if data.md == nil {
		return ""
	}

	if f, ok := data.md[metaName]; ok {
		return f()
	}

	return ""
}

// ServeDNS implements the plugin.Handler interface.
func (h Handler) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}

	zone := plugin.Zones(h.Zones).Matches(state.Name())
	if zone == "" {
		return plugin.NextOrFailure(h.Name(), h.Next, ctx, w, r)
	}

	for _, template := range h.Templates {
		data, match, fthrough := template.match(ctx, state)
		if !match {
			if !fthrough {
				return dns.RcodeServerFailure, nil
			}
			continue
		}

		templateMatchesCount.WithLabelValues(metrics.WithServer(ctx), data.Zone, metrics.WithView(ctx), data.Class, data.Type).Inc()

		if template.rcode == dns.RcodeServerFailure {
			return template.rcode, nil
		}

		msg := new(dns.Msg)
		msg.SetReply(r)
		msg.Authoritative = true
		msg.Rcode = template.rcode

		for _, answer := range template.answer {
			rr, err := executeRRTemplate(metrics.WithServer(ctx), metrics.WithView(ctx), "answer", answer, data)
			if err != nil {
				return dns.RcodeServerFailure, err
			}
			msg.Answer = append(msg.Answer, rr)
			if template.upstream != nil && (state.QType() == dns.TypeA || state.QType() == dns.TypeAAAA) && rr.Header().Rrtype == dns.TypeCNAME {
				if up, err := template.upstream.Lookup(ctx, state, rr.(*dns.CNAME).Target, state.QType()); err == nil && up != nil {
					msg.Truncated = up.Truncated
					msg.Answer = append(msg.Answer, up.Answer...)
				}
			}
		}
		for _, additional := range template.additional {
			rr, err := executeRRTemplate(metrics.WithServer(ctx), metrics.WithView(ctx), "additional", additional, data)
			if err != nil {
				return dns.RcodeServerFailure, err
			}
			msg.Extra = append(msg.Extra, rr)
		}
		for _, authority := range template.authority {
			rr, err := executeRRTemplate(metrics.WithServer(ctx), metrics.WithView(ctx), "authority", authority, data)
			if err != nil {
				return dns.RcodeServerFailure, err
			}
			msg.Ns = append(msg.Ns, rr)
		}

		if template.ederror != nil {
			msg = msg.SetEdns0(4096, true)
			ede := dns.EDNS0_EDE{InfoCode: template.ederror.code, ExtraText: template.ederror.reason}
			msg.IsEdns0().Option = append(msg.IsEdns0().Option, &ede)
		}

		w.WriteMsg(msg)
		return template.rcode, nil
	}

	return plugin.NextOrFailure(h.Name(), h.Next, ctx, w, r)
}

// Name implements the plugin.Handler interface.
func (h Handler) Name() string { return "template" }

func executeRRTemplate(server, view, section string, template *gotmpl.Template, data *templateData) (dns.RR, error) {
	buffer := &bytes.Buffer{}
	err := template.Execute(buffer, data)
	if err != nil {
		templateFailureCount.WithLabelValues(server, data.Zone, view, data.Class, data.Type, section, template.Tree.Root.String()).Inc()
		return nil, err
	}
	rr, err := dns.NewRR(buffer.String())
	if err != nil {
		templateRRFailureCount.WithLabelValues(server, data.Zone, view, data.Class, data.Type, section, template.Tree.Root.String()).Inc()
		return rr, err
	}
	return rr, nil
}

func newTemplate(name, text string) (*gotmpl.Template, error) {
	funcMap := gotmpl.FuncMap{
		"parseInt": strconv.ParseUint,
	}
	return gotmpl.New(name).Funcs(funcMap).Parse(text)
}

func (t template) match(ctx context.Context, state request.Request) (*templateData, bool, bool) {
	q := state.Req.Question[0]
	data := &templateData{md: metadata.ValueFuncs(ctx), Remote: state.IP()}

	zone := plugin.Zones(t.zones).Matches(state.Name())
	if zone == "" {
		return data, false, true
	}

	if t.qclass != dns.ClassANY && q.Qclass != dns.ClassANY && q.Qclass != t.qclass {
		return data, false, true
	}
	if t.qtype != dns.TypeANY && q.Qtype != dns.TypeANY && q.Qtype != t.qtype {
		return data, false, true
	}

	for _, regex := range t.regex {
		if !regex.MatchString(state.Name()) {
			continue
		}

		data.Zone = zone
		data.Regex = regex.String()
		data.Name = state.Name()
		data.Question = &q
		data.Message = state.Req
		if q.Qclass != dns.ClassANY {
			data.Class = dns.ClassToString[q.Qclass]
		} else {
			data.Class = dns.ClassToString[t.qclass]
		}
		if q.Qtype != dns.TypeANY {
			data.Type = dns.TypeToString[q.Qtype]
		} else {
			data.Type = dns.TypeToString[t.qtype]
		}

		matches := regex.FindStringSubmatch(state.Name())
		data.Match = make([]string, len(matches))
		data.Group = make(map[string]string)
		groupNames := regex.SubexpNames()
		for i, m := range matches {
			data.Match[i] = m
			data.Group[strconv.Itoa(i)] = m
		}
		for i, m := range matches {
			if len(groupNames[i]) > 0 {
				data.Group[groupNames[i]] = m
			}
		}

		return data, true, false
	}

	return data, false, t.fall.Through(state.Name())
}
