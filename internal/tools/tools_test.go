package tools

import "testing"

type fakeRunner struct {
	out  string
	err  error
	last []string
}

func (f *fakeRunner) Run(name string, args ...string) (string, error) {
	f.last = append([]string{name}, args...)
	return f.out, f.err
}

func TestParseADBDevices(t *testing.T) {
	out := "List of devices attached\n100.1.2.3:5555\tdevice\n10.0.0.9:5555\toffline\n"
	d := ParseADBDevices(out)
	if len(d) != 1 || d[0] != "100.1.2.3:5555" {
		t.Fatalf("want one connected device, got %v", d)
	}
}

func TestParseTailscaleRelayVsDirect(t *testing.T) {
	direct := "100.1.2.3   phone   linux   active; direct 1.2.3.4:41641"
	relay := "100.1.2.3   phone   linux   active; relay \"sin\""
	if s := ParseTailscalePeer(direct, "100.1.2.3"); !s.Found || s.Relay {
		t.Fatalf("expected direct, got %+v", s)
	}
	if s := ParseTailscalePeer(relay, "100.1.2.3"); !s.Found || !s.Relay {
		t.Fatalf("expected relay, got %+v", s)
	}
}

func TestConnectInvokesADB(t *testing.T) {
	f := &fakeRunner{out: "connected to 100.1.2.3:5555"}
	if _, err := Connect(f, "adb", "100.1.2.3", 5555); err != nil {
		t.Fatalf("connect: %v", err)
	}
	if len(f.last) != 3 || f.last[1] != "connect" || f.last[2] != "100.1.2.3:5555" {
		t.Fatalf("bad adb args: %v", f.last)
	}
}

func TestParsePing(t *testing.T) {
	relayMs, relay, ok := ParsePing("pong from galaxy-a06 (100.67.226.21) via DERP(sin) in 125ms")
	if !ok || relayMs != 125 || !relay {
		t.Fatalf("relay parse: ms=%d relay=%v ok=%v", relayMs, relay, ok)
	}
	dMs, dRelay, dOk := ParsePing("pong from x (100.1.2.3) via 1.2.3.4:41641 in 60ms")
	if !dOk || dMs != 60 || dRelay {
		t.Fatalf("direct parse: ms=%d relay=%v ok=%v", dMs, dRelay, dOk)
	}
	if _, _, ok := ParsePing("direct connection not established"); ok {
		t.Fatalf("expected no-pong to be ok=false")
	}
}

func TestTcpipInvokesADB(t *testing.T) {
	f := &fakeRunner{out: "restarting in TCP mode port: 5555"}
	if _, err := Tcpip(f, "adb", "100.1.2.3:34171", 5555); err != nil {
		t.Fatalf("tcpip: %v", err)
	}
	want := []string{"adb", "-s", "100.1.2.3:34171", "tcpip", "5555"}
	if len(f.last) != len(want) {
		t.Fatalf("args = %v", f.last)
	}
	for i := range want {
		if f.last[i] != want[i] {
			t.Fatalf("args[%d]=%q want %q (%v)", i, f.last[i], want[i], f.last)
		}
	}
}
