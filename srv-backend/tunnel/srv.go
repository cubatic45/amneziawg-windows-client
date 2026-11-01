// SPDX-License-Identifier: MIT
package tunnel

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/amnezia-vpn/amneziawg-windows/v3/conf"
)

// srvUpdate only changes existing peers, without replacing keys, routes or counters.
func srvUpdate(key conf.Key, endpoint conf.Endpoint) string {
	return fmt.Sprintf("public_key=%s\nupdate_only=true\nendpoint=%s\n", key.HexString(), endpoint.String())
}

func startSRVRefresh(config *conf.Config, last map[conf.Key]conf.Endpoint, apply func(string) error) func() {
	stop := make(chan struct{})
	done := make(chan struct{})
	if len(last) == 0 {
		close(done)
		return func() {}
	}
	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				for _, peer := range config.Peers {
					if !peer.Endpoint.IsSRV() {
						continue
					}
					endpoint, err := peer.Endpoint.Resolve()
					select {
					case <-stop:
						return
					default:
					}
					if err != nil {
						log.Printf("SRV refresh failed for %s; keeping current endpoint: %v", peer.Endpoint.Host, err)
						continue
					}
					if endpoint == last[peer.PublicKey] {
						continue
					}
					if err := apply(srvUpdate(peer.PublicKey, endpoint)); err != nil {
						log.Printf("SRV endpoint update failed: %v", err)
						continue
					}
					last[peer.PublicKey] = endpoint
					log.Printf("SRV endpoint refreshed: %s -> %s", strings.TrimPrefix(peer.Endpoint.Host, "srv://"), endpoint.String())
				}
			}
		}
	}()
	// Stop before the service closes the device; in-flight DNS has a 10s deadline.
	return func() { close(stop); <-done }
}
