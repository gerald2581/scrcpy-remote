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
