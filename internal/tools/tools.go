package tools

import (
	"fmt"
	"strings"
)

// ParseADBDevices returns serials in "device" state from `adb devices` output.
func ParseADBDevices(out string) []string {
	var res []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of devices") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "device" {
			res = append(res, fields[0])
		}
	}
	return res
}

// PeerStatus is a peer's reachability derived from `tailscale status`.
type PeerStatus struct {
	Found bool
	Relay bool // true if traffic goes via a DERP relay (higher latency) rather than direct
}

// ParseTailscalePeer scans `tailscale status` output for the line of the given IP.
func ParseTailscalePeer(out, ip string) PeerStatus {
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, ip) {
			return PeerStatus{Found: true, Relay: strings.Contains(line, "relay ")}
		}
	}
	return PeerStatus{}
}

// Pair runs `adb pair ip:port` with the code (Android 11+).
func Pair(r Runner, adb, ip string, pairPort int, code string) (string, error) {
	return r.Run(adb, "pair", fmt.Sprintf("%s:%d", ip, pairPort), code)
}

// Connect runs `adb connect ip:port`.
func Connect(r Runner, adb, ip string, adbPort int) (string, error) {
	return r.Run(adb, "connect", fmt.Sprintf("%s:%d", ip, adbPort))
}
