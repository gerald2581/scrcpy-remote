package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"scrcpy-remote/internal/devices"
	"scrcpy-remote/internal/scrcpy"
	"scrcpy-remote/internal/tools"
)

// Server holds the API dependencies. LaunchFn is injectable so tests don't spawn scrcpy.
type Server struct {
	ConfigPath string
	Runner     tools.Runner
	ADBPath    string
	ScrcpyPath string
	LaunchFn   func(args []string) error
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
	return mux
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
