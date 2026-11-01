# Windows SRV test build

This unofficial x64 build adds opt-in DNS SRV discovery to the standalone
AmneziaWG Windows client. No upstream Windows pull request has been submitted.

## Installation and configuration

1. Export or back up existing tunnel configurations.
2. Install the x64 MSI as an administrator on Windows 10/11 x64. The package
   uses the standard AmneziaWG installation and upgrade identity. It replaces
   the installed client; it is not a separate side-by-side application.
3. In a peer, replace the fixed endpoint with the complete SRV owner:

   ```ini
   Endpoint = srv://_wireguard._udp.vpn.example.com
   ```

4. Leave keys, AllowedIPs and Amnezia obfuscation settings appropriate for
   your server. Activate the tunnel and inspect its log.

Publish `_wireguard._udp.vpn.example.com` as a DNS SRV record whose target has
an A/AAAA record. Its port is the actual externally mapped UDP port. The
Cloudflare target must be DNS-only, not an HTTP proxy. The SRV URL has no port,
path, query or fragment. Ordinary `hostname:port`, IPv4 and IPv6 endpoints keep
their existing behavior.

The portable ZIP contains `amneziawg.exe`, `awg.exe` and the official signed
`wintun.dll`. Extract all files to the same directory and run the GUI as an
administrator. It uses the same Windows services as the installed client.
Do not run installed and portable copies simultaneously.

## Behavior and limits

The Windows system resolver queries the exact SRV owner, applies RFC 2782
priority/weight ordering, then resolves the targets to IPv4/IPv6. Address
resolution failure falls through to another SRV target. This is not handshake
health-based failover. A target of `.` means unavailable; there is no guessed
fallback port.

SRV endpoints are resolved before startup and refreshed every 60 seconds while
the tunnel is active. Windows DNS caching/record TTL can delay a DNS change.
A changed IP or port is applied through an endpoint-only UAPI update. A failed
refresh retains the previous working configuration; an unchanged DNS result
is not reapplied over a peer's learned roaming endpoint. Config editing and
export retain the original SRV URL.

System DNS must be reachable both before startup and during refresh. VPN-only
DNS cannot recover a disconnected tunnel on its own. Full-tunnel firewall/DNS
settings may affect refresh reachability. The existing socket-to-interface
binding continues to route the tunnel transport through a physical interface.

The GUI and MSI are unsigned community test builds. Windows may show an
unknown publisher/SmartScreen prompt. Wintun is copied without modification
from the upstream signed distribution; no driver signature enforcement is
disabled. The normal updater rejects unofficial builds. To roll back, stop
all tunnels, uninstall this build, and install the official client. Restore
fixed `host:port` endpoints before using a client without SRV support.

## Build provenance and verification

`srv-backend/` is a source snapshot of
`amnezia-vpn/amneziawg-windows` tag `v3.1.20260814`
(commit `e90531d15802cb976773f3b63443bc281f738ca3`), with changes to config
parsing/export, DNS resolution, and the tunnel service. A local Go module
replacement ensures the GUI uses this source. Original copyright/license
headers are retained.

`.github/workflows/srv-build.yml` runs deterministic resolver/config tests,
a live Windows system-DNS SRV lookup, endpoint-refresh tests, and the official
GUI/MSI build scripts on a Windows runner. Downloads retain upstream SHA-256
verification. Set `SRV_X64_ONLY=1` to build only x64. Artifact SHA-256 hashes
are included with each build. A real VPN handshake and traffic test on the
user's Windows machine is still required.
