package scrcpy

import "fmt"

// Serial builds the adb serial "ip:port" used with scrcpy's -s flag.
func Serial(ip string, adbPort int) string {
	return fmt.Sprintf("%s:%d", ip, adbPort)
}

// presets maps a preset name to its scrcpy flag tail (after -s <serial>).
var presets = map[string][]string{
	"balanced": {"--video-codec=h264", "--max-size", "1024", "--video-bit-rate", "2M", "--max-fps", "30", "--no-audio"},
	"fast":     {"--video-codec=h264", "--max-size", "800", "--video-bit-rate", "1500K", "--max-fps", "20", "--no-audio"},
	"hd":       {"--video-codec=h264", "--max-size", "1600", "--video-bit-rate", "8M", "--max-fps", "60"},
	// lagfree targets relay (DERP/peer-relay) links: tiny --video-buffer damps
	// network jitter with negligible added latency. See docs/superpowers research.
	"lagfree":  {"--video-codec=h264", "--max-size", "640", "--video-bit-rate", "1M", "--max-fps", "15", "--no-audio", "--video-buffer=50"},
}

// Flags returns the scrcpy argument slice for a preset targeting a serial.
// Unknown presets fall back to "balanced".
func Flags(preset, serial string) []string {
	tail, ok := presets[preset]
	if !ok {
		tail = presets["balanced"]
	}
	args := []string{"-s", serial}
	return append(args, tail...)
}
