package dns

import (
	"fmt"
	"net"

	"github.com/miekg/dns"
)

type Address struct {
	Address string
	Port    uint16
}

func (a Address) String() string {
	return fmt.Sprintf("%s:%d", a.Address, a.Port)
}

type Lookup interface {
	LookupSRV(name string) ([]net.SRV, error)
	LookupA(name string) (string, error)
}

func NewLookup(serverString string) *lookup {
	l := new(lookup)
	l.serverString = serverString
	return l
}

type lookup struct {
	serverString string
}

func (l *lookup) LookupSRV(name string) ([]net.SRV, error) {
	var srvs = make([]net.SRV, 0)
	answer, err := l.lookupType(name, "SRV")
	if err != nil {
		return srvs, err
	}
	return l.parseSRVAnswer(answer)
}

func (l *lookup) LookupA(name string) (string, error) {
	answer, err := l.lookupType(name, "A")
	if err != nil {
		return "", err
	}
	return l.parseAAnswer(answer)
}

func (l *lookup) parseSRVAnswer(answer *dns.Msg) ([]net.SRV, error) {
	var srvs = make([]net.SRV, 0)
	for _, v := range answer.Answer {
		if srv, ok := v.(*dns.SRV); ok {
			srvs = append(srvs, net.SRV{
				Priority: srv.Priority,
				Weight:   srv.Weight,
				Port:     srv.Port,
				Target:   srv.Target,
			})
		}
	}
	return srvs, nil
}

func (l *lookup) parseAAnswer(answer *dns.Msg) (string, error) {
	if len(answer.Answer) == 0 {
		return "", fmt.Errorf("Answer Empty")
	}
	if a, ok := answer.Answer[0].(*dns.A); ok {
		return a.A.String(), nil
	}
	return "", fmt.Errorf("Could not parse A record")
}

func (l *lookup) lookupType(name string, recordType string) (*dns.Msg, error) {
	// try a connection with a udp connection first
	return l.lookup(name, recordType, "")
}

func (l *lookup) lookup(name string, recordType string, connType string) (*dns.Msg, error) {
	qType, ok := dns.StringToType[recordType]
	if !ok {
		return nil, fmt.Errorf("Invalid type '%s'", recordType)
	}
	name = dns.Fqdn(name)

	client := &dns.Client{Net: connType}

	msg := &dns.Msg{}
	msg.SetQuestion(name, qType)

	response, _, err := client.Exchange(msg, l.serverString)
	if err != nil {
		if connType == "" {
			// retry lookup with a tcp connection
			return l.lookup(name, recordType, "tcp")
		} else {
			return nil, fmt.Errorf("Couldn't resolve name '%s'", name)
		}
	}

	if msg.Id != response.Id {
		return nil, fmt.Errorf("DNS ID mismatch, request: %d, response: %d", msg.Id, response.Id)
	}

	return response, nil
}
