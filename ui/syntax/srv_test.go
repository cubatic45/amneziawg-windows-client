package syntax

import "testing"

func TestSRVEndpointHighlighting(t *testing.T) {
	for _, endpoint := range []string{"srv://_wireguard._udp.example.com", "srv://_awg._udp.example.com.", "192.0.2.1:51820", "[2001:db8::1]:1234"} {
		config := "[Peer]\nEndpoint = " + endpoint + "\n"
		for _, span := range highlightConfig(config) {
			if span.t == highlightError {
				t.Errorf("editor rejected %q", endpoint)
			}
		}
	}
	for _, endpoint := range []string{"srv://example.com", "srv://_wg._udp.example.com:1234"} {
		found := false
		for _, span := range highlightConfig("[Peer]\nEndpoint = " + endpoint + "\n") {
			if span.t == highlightError {
				found = true
			}
		}
		if !found {
			t.Errorf("editor accepted %q", endpoint)
		}
	}
}
