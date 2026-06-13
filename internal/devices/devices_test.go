package devices

import (
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	in := Config{Port: 8787, Devices: []Device{{ID: "a", Name: "Phone", IP: "100.1.2.3", ADBPort: 5555, PairPort: 37000}}}
	if err := Save(p, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	out, err := Load(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if out.Port != 8787 || len(out.Devices) != 1 || out.Devices[0].IP != "100.1.2.3" {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}

func TestLoadMissingReturnsDefault(t *testing.T) {
	out, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatalf("expected default, got err: %v", err)
	}
	if out.Port != 8787 {
		t.Fatalf("expected default port 8787, got %d", out.Port)
	}
}

func TestAddOrUpdateAndDelete(t *testing.T) {
	c := Config{Port: 8787}
	c = c.AddOrUpdate(Device{ID: "x", Name: "A", IP: "100.0.0.1", ADBPort: 5555})
	c = c.AddOrUpdate(Device{ID: "x", Name: "A2", IP: "100.0.0.1", ADBPort: 5555})
	if len(c.Devices) != 1 || c.Devices[0].Name != "A2" {
		t.Fatalf("update should replace by ID: %+v", c.Devices)
	}
	c = c.Delete("x")
	if len(c.Devices) != 0 {
		t.Fatalf("delete failed: %+v", c.Devices)
	}
}
