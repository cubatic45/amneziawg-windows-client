// SPDX-License-Identifier: MIT
package conf

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// ValidSRVEndpoint accepts an explicit UDP SRV owner, without a port or URL path.
func ValidSRVEndpoint(value string) bool {
	if !strings.HasPrefix(value, "srv://") {
		return false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(value, "srv://"), ".")
	if len(name) > 253 {
		return false
	}
	labels := strings.Split(name, ".")
	if len(labels) < 3 || len(labels[0]) < 2 || labels[0][0] != '_' || labels[1] != "_udp" {
		return false
	}
	for i, label := range labels {
		if i < 2 {
			label = strings.TrimPrefix(label, "_")
		}
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}

func (e Endpoint) IsSRV() bool { return strings.HasPrefix(e.Host, "srv://") }

func (e Endpoint) Resolve() (Endpoint, error) {
	if !e.IsSRV() {
		ip, err := resolveHostname(e.Host)
		return Endpoint{ip, e.Port}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return resolveSRV(ctx, e, net.DefaultResolver.LookupSRV, func(ctx context.Context, host string) (string, error) {
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return "", err
		}
		// Match the ordinary Windows endpoint preference for IPv4, then IPv6.
		for _, ip := range ips {
			if ip.IP.To4() != nil {
				return ip.String(), nil
			}
		}
		if len(ips) > 0 {
			return ips[0].String(), nil
		}
		return "", fmt.Errorf("no address for SRV target %s", host)
	})
}

type srvLookup func(context.Context, string, string, string) (string, []*net.SRV, error)
type addressLookup func(context.Context, string) (string, error)

func resolveSRV(ctx context.Context, e Endpoint, lookup srvLookup, address addressLookup) (Endpoint, error) {
	if !ValidSRVEndpoint(e.Host) {
		return Endpoint{}, fmt.Errorf("invalid SRV endpoint %q", e.Host)
	}
	// Empty service/proto makes LookupSRV query this exact owner. Go's resolver
	// orders records by RFC 2782 priority and randomized weight on Windows too.
	_, records, err := lookup(ctx, "", "", strings.TrimPrefix(e.Host, "srv://"))
	if err != nil {
		return Endpoint{}, err
	}
	if len(records) == 0 {
		return Endpoint{}, fmt.Errorf("no SRV records for %s", e.Host)
	}
	for _, record := range records {
		if record == nil {
			return Endpoint{}, fmt.Errorf("invalid nil SRV record")
		}
		if record.Target == "." {
			return Endpoint{}, fmt.Errorf("SRV service unavailable")
		}
		if record.Port == 0 || record.Target == "" {
			continue
		}
		ip, lookupErr := address(ctx, record.Target)
		if lookupErr != nil {
			err = lookupErr
			continue
		}
		host := strings.SplitN(ip, "%", 2)[0]
		if net.ParseIP(host) == nil {
			err = fmt.Errorf("invalid resolved address %q", ip)
			continue
		}
		return Endpoint{ip, record.Port}, nil
	}
	if err == nil {
		err = fmt.Errorf("no usable SRV targets for %s", e.Host)
	}
	return Endpoint{}, err
}
