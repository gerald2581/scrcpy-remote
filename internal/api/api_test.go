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

// recRunner records every Run call (and always succeeds) to assert call order.
type recRunner struct{ calls [][]string }

func (r *recRunner) Run(name string, args ...string) (string, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	return "ok", nil
}

func TestBootstrapPairsConnectsTcpipAndSaves5555(t *testing.T) {
	s := newServer(t)
	rec := &recRunner{}
	s.Runner = rec
	body := `{"id":"x","name":"P","ip":"100.1.2.3","wdPort":34171,"pairPort":45697,"pairCode":"393314"}`
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest("POST", "/api/bootstrap", strings.NewReader(body)))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body)
	}
	if len(rec.calls) != 4 {
		t.Fatalf("calls = %v", rec.calls)
	}
	if rec.calls[0][1] != "pair" || rec.calls[0][2] != "100.1.2.3:45697" || rec.calls[0][3] != "393314" {
		t.Fatalf("pair call wrong: %v", rec.calls[0])
	}
	if rec.calls[1][1] != "connect" || rec.calls[1][2] != "100.1.2.3:34171" {
		t.Fatalf("connect wdPort wrong: %v", rec.calls[1])
	}
	if rec.calls[2][1] != "-s" || rec.calls[2][3] != "tcpip" || rec.calls[2][4] != "5555" {
		t.Fatalf("tcpip wrong: %v", rec.calls[2])
	}
	if rec.calls[3][1] != "connect" || rec.calls[3][2] != "100.1.2.3:5555" {
		t.Fatalf("connect 5555 wrong: %v", rec.calls[3])
	}
	w2 := httptest.NewRecorder()
	s.Handler().ServeHTTP(w2, httptest.NewRequest("GET", "/api/devices", nil))
	if !strings.Contains(w2.Body.String(), `"adbPort":5555`) {
		t.Fatalf("device not saved at 5555: %s", w2.Body)
	}
}

func TestBootstrapSkipsPairWhenNoCode(t *testing.T) {
	s := newServer(t)
	rec := &recRunner{}
	s.Runner = rec
	body := `{"id":"y","name":"Q","ip":"100.9.9.9","wdPort":40000}`
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest("POST", "/api/bootstrap", strings.NewReader(body)))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body)
	}
	if len(rec.calls) != 3 || rec.calls[0][1] != "connect" {
		t.Fatalf("expected no pair, got %v", rec.calls)
	}
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
