package tunnel

import (
	"errors"
	"github.com/amnezia-vpn/amneziawg-windows/v3/conf"
	"strings"
	"testing"
)

func TestSRVEndpointRefresh(t *testing.T) {
	key := conf.Key{1}
	original := conf.Endpoint{Host: "192.0.2.1", Port: 1111}
	next := conf.Endpoint{Host: "192.0.2.2", Port: 2222}
	last := map[conf.Key]conf.Endpoint{key: original}
	calls := 0
	apply := func(uapi string) error {
		calls++
		for _, expected := range []string{"public_key=" + key.HexString(), "update_only=true", "endpoint=192.0.2.2:2222"} {
			if !strings.Contains(uapi, expected) {
				t.Fatalf("missing %q in %q", expected, uapi)
			}
		}
		if strings.Contains(uapi, "replace_peers") || strings.Contains(uapi, "private_key") {
			t.Fatal("destructive refresh")
		}
		return nil
	}
	if changed, err := applySRVEndpoint(key, original, last, apply); changed || err != nil || calls != 0 {
		t.Fatal("unchanged DNS overwrote roaming endpoint")
	}
	if changed, err := applySRVEndpoint(key, next, last, func(string) error { return errors.New("backend unavailable") }); changed || err == nil || last[key] != original {
		t.Fatal("failed update changed cache")
	}
	if changed, err := applySRVEndpoint(key, next, last, apply); !changed || err != nil || calls != 1 || last[key] != next {
		t.Fatal("refresh did not update cache")
	}
	if changed, err := applySRVEndpoint(key, next, last, apply); changed || err != nil || calls != 1 {
		t.Fatal("duplicate update")
	}
	// A port-only change must also update without replacing peers.
	portOnly := conf.Endpoint{Host: next.Host, Port: 3333}
	if changed, err := applySRVEndpoint(key, portOnly, last, func(uapi string) error {
		if !strings.Contains(uapi, "endpoint=192.0.2.2:3333") {
			t.Fatal("port change lost")
		}
		return nil
	}); !changed || err != nil {
		t.Fatal("port-only refresh failed")
	}
}
