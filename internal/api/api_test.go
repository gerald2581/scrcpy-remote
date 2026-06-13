package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRunner struct{ out string }

func (f fakeRunner) Run(name string, args ...string) (string, error) { return f.out, nil }

func newServer(t *testing.T) *Server {
	return &Server{
		ConfigPath: filepath.Join(t.TempDir(), "config.json"),
		Runner:     fakeRunner{out: "connected to 100.1.2.3:5555"},
		ADBPath:    "adb",
		ScrcpyPath: "scrcpy",
		LaunchFn:   func(args []string) error { return nil },
	}
}

func TestDevicesCRUD(t *testing.T) {
	s := newServer(t)
	body := `{"id":"x","name":"Phone","ip":"100.1.2.3","adbPort":5555,"pairPort":37000}`
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/api/devices", strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("post status %d: %s", rec.Code, rec.Body)
	}
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/api/devices", nil))
	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			Devices []map[string]any `json:"devices"`
		} `json:"data"`
	}
	json.Unmarshal(rec.Body.Bytes(), &env)
	if !env.OK || len(env.Data.Devices) != 1 {
		t.Fatalf("expected 1 device, body=%s", rec.Body)
	}
}

func TestLaunchUsesPreset(t *testing.T) {
	s := newServer(t)
	var gotArgs []string
	s.LaunchFn = func(args []string) error { gotArgs = args; return nil }
	body := `{"ip":"100.1.2.3","adbPort":5555,"preset":"fast"}`
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/api/launch", strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("launch status %d: %s", rec.Code, rec.Body)
	}
	if strings.Join(gotArgs, " ") == "" || !strings.Contains(strings.Join(gotArgs, " "), "1500K") {
		t.Fatalf("expected fast preset flags, got %v", gotArgs)
	}
}

// multiRunner returns different output for `adb devices` vs `tailscale status`.
type multiRunner struct{}

func (multiRunner) Run(name string, args ...string) (string, error) {
	if len(args) > 0 && args[0] == "devices" {
		return "List of devices attached\n100.1.2.3:5555\tdevice\n", nil
	}
	if len(args) > 0 && args[0] == "status" {
		return "100.1.2.3   phone   linux   active; relay \"sin\"", nil
	}
	return "", nil
}

func TestStatusReflectsAdbAndTailscale(t *testing.T) {
	s := newServer(t)
	s.Runner = multiRunner{}
	s.TailscalePath = "tailscale"
	// register a device whose serial matches the faked adb output
	s.Handler().ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", "/api/devices", strings.NewReader(`{"id":"x","name":"P","ip":"100.1.2.3","adbPort":5555}`)))

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/api/status", nil))
	var env struct {
		OK   bool `json:"ok"`
		Data []struct {
			ID        string `json:"id"`
			Connected bool   `json:"connected"`
			TSFound   bool   `json:"tsFound"`
			TSRelay   bool   `json:"tsRelay"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v body=%s", err, rec.Body)
	}
	if !env.OK || len(env.Data) != 1 {
		t.Fatalf("want 1 status, body=%s", rec.Body)
	}
	d := env.Data[0]
	if !d.Connected || !d.TSFound || !d.TSRelay {
		t.Fatalf("status wrong: %+v", d)
	}
}
