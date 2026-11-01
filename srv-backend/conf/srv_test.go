package conf

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"testing"
)

func TestSRVEndpointParsing(t *testing.T) {
	for _, value := range []string{"srv://_wireguard._udp.vpn.example.com", "srv://_awg._udp.example.com."} {
		endpoint, err := parseEndpoint(value)
		if err != nil || !endpoint.IsSRV() || endpoint.String() != value {
			t.Fatalf("%q: %v, %v", value, endpoint, err)
		}
	}
	for _, value := range []string{"srv://example.com", "srv://_wg._tcp.example.com", "srv://_wg._udp.example.com:123", "srv://_wg._udp.example.com/path", "srv://_wg._udp..com", "srv://_._udp.example.com", "srv://_wg._udp.-bad.com", "srv://_wg._udp.example.com?x"} {
		if _, err := parseEndpoint(value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
	for _, value := range []string{"example.com:51820", "192.0.2.1:1234", "[2001:db8::1]:1234"} {
		endpoint, err := parseEndpoint(value)
		if err != nil || endpoint.IsSRV() || endpoint.String() != value {
			t.Fatalf("regression %q: %v", value, err)
		}
	}
}
func TestSRVResolution(t *testing.T) {
	lookup := func(ctx context.Context, service, proto, name string) (string, []*net.SRV, error) {
		if service != "" || proto != "" || name != "_wireguard._udp.example.com" {
			t.Fatal("incorrect owner")
		}
		return "", []*net.SRV{{Target: "bad.example.com.", Port: 1}, {Target: "good.example.com.", Port: 9783}}, nil
	}
	address := func(ctx context.Context, host string) (string, error) {
		if host == "bad.example.com." {
			return "", errors.New("unresolvable")
		}
		return "2001:db8::1", nil
	}
	got, err := resolveSRV(context.Background(), Endpoint{Host: "srv://_wireguard._udp.example.com"}, lookup, address)
	if err != nil || got.String() != "[2001:db8::1]:9783" {
		t.Fatalf("%v, %v", got, err)
	}
}
func TestSRVUnavailableAndErrors(t *testing.T) {
	cases := []struct {
		name    string
		records []*net.SRV
		err     error
	}{
		{"unavailable", []*net.SRV{{Target: ".", Port: 0}}, nil},
		{"empty", nil, nil}, {"dns error", nil, errors.New("DNS unavailable")},
		{"nil record", []*net.SRV{nil}, nil},
		{"zero port", []*net.SRV{{Target: "vpn.example.com.", Port: 0}}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lookup := func(context.Context, string, string, string) (string, []*net.SRV, error) {
				return "", tc.records, tc.err
			}
			_, err := resolveSRV(context.Background(), Endpoint{Host: "srv://_wg._udp.example.com"}, lookup, func(context.Context, string) (string, error) { t.Fatal("unexpected address lookup"); return "", nil })
			if err == nil {
				t.Fatal("expected failure")
			}
		})
	}
}
func TestSRVExportAndLiveConfigPreserveDiscovery(t *testing.T) {
	value := "srv://_wireguard._udp.example.com"
	config, err := FromWgQuick(strings.Replace(testInput, "192.95.5.67:1234", value, 1), "srv-test")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(config.ToWgQuick(), "Endpoint = "+value) {
		t.Fatal("lost SRV endpoint")
	}
	data := "public_key=" + config.Peers[0].PublicKey.HexString() + "\nendpoint=192.0.2.1:9783\nerrno=0\n\n"
	live, err := FromUAPI(strings.NewReader(data), config)
	if err != nil {
		t.Fatal(err)
	}
	if live.Peers[0].Endpoint.String() != value {
		t.Fatal("lost discovery URL")
	}
}
func TestSRVLiveDNS(t *testing.T) {
	if os.Getenv("SRV_LIVE_DNS") != "1" {
		t.Skip("set SRV_LIVE_DNS=1 for system DNS integration")
	}
	endpoint, err := (Endpoint{Host: "srv://_sip._udp.sip.linphone.org"}).Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if endpoint.Port != 5060 || net.ParseIP(endpoint.Host) == nil {
		t.Fatalf("invalid endpoint: %v", endpoint)
	}
	t.Logf("Windows system DNS SRV: %s", endpoint.String())
}
