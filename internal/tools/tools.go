package tools

import (
	"fmt"
	"strconv"
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

// Tcpip restarts adbd on the device in TCP mode on a fixed port:
// `adb -s <serial> tcpip <port>`. After this the device listens on that port on all
// interfaces, surviving network changes until the device reboots.
func Tcpip(r Runner, adb, serial string, port int) (string, error) {
	return r.Run(adb, "-s", serial, "tcpip", fmt.Sprintf("%d", port))
}

// ParsePing extracts the round-trip latency (ms) and whether it is relayed (DERP) from a
// `tailscale ping` pong line like `pong from x (ip) via DERP(sin) in 125ms`.
func ParsePing(out string) (ms int, relay bool, ok bool) {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, " in ") || !strings.HasSuffix(line, "ms") {
			continue
		}
		f := strings.Fields(line)
		n, err := strconv.Atoi(strings.TrimSuffix(f[len(f)-1], "ms"))
		if err != nil {
			continue
		}
		return n, strings.Contains(line, "DERP"), true
	}
	return 0, false, false
}

// Ping runs a single `tailscale ping` and returns latency in ms + whether it is relayed.
func Ping(r Runner, ts, ip string) (ms int, relay bool, ok bool) {
	out, _ := r.Run(ts, "ping", "--c", "1", ip)
	return ParsePing(out)
}
