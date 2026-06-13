package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"scrcpy-remote/internal/devices"
	"scrcpy-remote/internal/scrcpy"
	"scrcpy-remote/internal/tools"
)

// FixedADBPort is the stable adb port a device is switched to via `adb tcpip`, so it stays
// reachable across network changes (until the device reboots).
const FixedADBPort = 5555

const (
	bootstrapConnectRetries = 4
	bootstrapRetryDelay     = 1500 * time.Millisecond
)

// Server holds the API dependencies. LaunchFn is injectable so tests don't spawn scrcpy.
type Server struct {
	ConfigPath    string
	Runner        tools.Runner
	ADBPath       string
	ScrcpyPath    string
	TailscalePath string
	LaunchFn      func(args []string) error
}

func writeJSON(w http.ResponseWriter, ok bool, data any, errMsg string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": ok, "data": data, "error": errMsg})
}

// Handler returns the API mux (mounted under /api by main).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/devices", s.devices)
	mux.HandleFunc("/api/connect", s.connect)
	mux.HandleFunc("/api/launch", s.launch)
	mux.HandleFunc("/api/status", s.status)
	mux.HandleFunc("/api/bootstrap", s.bootstrap)
	return mux
}

// bootstrap pairs (optional) + connects to the dynamic Wireless-Debugging port, switches the
// device to the fixed port via `adb tcpip`, reconnects on the fixed port, and saves the device
// at FixedADBPort so future connects survive network changes.
func (s *Server) bootstrap(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		IP       string `json:"ip"`
		WDPort   int    `json:"wdPort"`
		PairPort int    `json:"pairPort"`
		PairCode string `json:"pairCode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IP == "" || req.WDPort == 0 {
		writeJSON(w, false, nil, "ip and wdPort are required")
		return
	}
	if req.PairPort != 0 && req.PairCode != "" {
		if out, err := tools.Pair(s.Runner, s.ADBPath, req.IP, req.PairPort, req.PairCode); err != nil {
			writeJSON(w, false, out, "pair failed: "+err.Error())
			return
		}
	}
	if out, err := tools.Connect(s.Runner, s.ADBPath, req.IP, req.WDPort); err != nil {
		writeJSON(w, false, out, "connect (WD port) failed: "+err.Error())
		return
	}
	wdSerial := scrcpy.Serial(req.IP, req.WDPort)
	if out, err := tools.Tcpip(s.Runner, s.ADBPath, wdSerial, FixedADBPort); err != nil {
		writeJSON(w, false, out, "tcpip failed: "+err.Error())
		return
	}
	var cerr error
	for i := 0; i < bootstrapConnectRetries; i++ {
		if i > 0 {
			time.Sleep(bootstrapRetryDelay) // adbd needs a moment to restart in TCP mode
		}
		if _, cerr = tools.Connect(s.Runner, s.ADBPath, req.IP, FixedADBPort); cerr == nil {
			break
		}
	}
	if cerr != nil {
		writeJSON(w, false, nil, "connect (fixed port) failed: "+cerr.Error())
		return
	}
	cfg, err := devices.Load(s.ConfigPath)
	if err != nil {
		writeJSON(w, false, nil, err.Error())
		return
	}
	cfg = cfg.AddOrUpdate(devices.Device{ID: req.ID, Name: req.Name, IP: req.IP, ADBPort: FixedADBPort})
	if err := devices.Save(s.ConfigPath, cfg); err != nil {
		writeJSON(w, false, nil, err.Error())
		return
	}
	writeJSON(w, true, map[string]int{"adbPort": FixedADBPort}, "")
}

// devStatus is the live status of one configured device.
type devStatus struct {
	ID        string `json:"id"`
	Connected bool   `json:"connected"` // adb sees the serial as "device"
	TSFound   bool   `json:"tsFound"`   // peer present in `tailscale status`
	TSRelay   bool   `json:"tsRelay"`   // reachable via DERP relay (higher latency) vs direct
}

func contains(list []string, v string) bool {
	for _, e := range list {
		if e == v {
			return true
		}
	}
	return false
}

// status returns per-device live status from `adb devices` + `tailscale status`.
func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	cfg, err := devices.Load(s.ConfigPath)
	if err != nil {
		writeJSON(w, false, nil, err.Error())
		return
	}
	adbOut, _ := s.Runner.Run(s.ADBPath, "devices")
	connected := tools.ParseADBDevices(adbOut)
	tsOut, _ := s.Runner.Run(s.TailscalePath, "status")

	out := make([]devStatus, 0, len(cfg.Devices))
	for _, d := range cfg.Devices {
		peer := tools.ParseTailscalePeer(tsOut, d.IP)
		out = append(out, devStatus{
			ID:        d.ID,
			Connected: contains(connected, scrcpy.Serial(d.IP, d.ADBPort)),
			TSFound:   peer.Found,
			TSRelay:   peer.Relay,
		})
	}
	writeJSON(w, true, out, "")
}

func (s *Server) devices(w http.ResponseWriter, r *http.Request) {
	cfg, err := devices.Load(s.ConfigPath)
	if err != nil {
		writeJSON(w, false, nil, err.Error())
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, true, cfg, "")
	case http.MethodPost:
		var d devices.Device
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil || d.IP == "" {
			writeJSON(w, false, nil, "invalid device")
			return
		}
		cfg = cfg.AddOrUpdate(d)
		if err := devices.Save(s.ConfigPath, cfg); err != nil {
			writeJSON(w, false, nil, err.Error())
			return
		}
		writeJSON(w, true, cfg, "")
	default:
		writeJSON(w, false, nil, "method not allowed")
	}
}

func (s *Server) connect(w http.ResponseWriter, r *http.Request) {
	var d devices.Device
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil || d.IP == "" {
		writeJSON(w, false, nil, "invalid device")
		return
	}
	out, err := tools.Connect(s.Runner, s.ADBPath, d.IP, d.ADBPort)
	if err != nil {
		writeJSON(w, false, out, err.Error())
		return
	}
	writeJSON(w, true, out, "")
}

func (s *Server) launch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IP      string `json:"ip"`
		ADBPort int    `json:"adbPort"`
		Preset  string `json:"preset"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IP == "" {
		writeJSON(w, false, nil, "invalid request")
		return
	}
	args := scrcpy.Flags(req.Preset, scrcpy.Serial(req.IP, req.ADBPort))
	if err := s.LaunchFn(args); err != nil {
		writeJSON(w, false, nil, err.Error())
		return
	}
	writeJSON(w, true, strings.Join(args, " "), "")
}
