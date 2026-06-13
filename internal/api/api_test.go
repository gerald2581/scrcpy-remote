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
	if strings.Join(gotArgs, " ") == "" || !strings.Contains(strings.Join(gotArgs, " "), "1.5M") {
		t.Fatalf("expected fast preset flags, got %v", gotArgs)
	}
}
