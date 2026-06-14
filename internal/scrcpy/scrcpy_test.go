package scrcpy

import (
	"strings"
	"testing"
)

func TestSerial(t *testing.T) {
	if got := Serial("100.1.2.3", 5555); got != "100.1.2.3:5555" {
		t.Fatalf("got %q", got)
	}
}

func TestFlagsKnownPresets(t *testing.T) {
	cases := map[string]string{"balanced": "2M", "fast": "1500K", "hd": "8M", "lagfree": "1M"}
	for preset, wantBitrate := range cases {
		f := Flags(preset, "100.1.2.3:5555")
		joined := strings.Join(f, " ")
		if !strings.Contains(joined, "-s 100.1.2.3:5555") {
			t.Fatalf("%s: missing serial: %v", preset, f)
		}
		if !strings.Contains(joined, "--video-bit-rate "+wantBitrate) {
			t.Fatalf("%s: want bitrate %s in %v", preset, wantBitrate, f)
		}
	}
}

func TestFlagsUnknownPresetFallsBackToBalanced(t *testing.T) {
	f := strings.Join(Flags("bogus", "x:1"), " ")
	if !strings.Contains(f, "--video-bit-rate 2M") {
		t.Fatalf("unknown preset should fall back to balanced: %s", f)
	}
}
