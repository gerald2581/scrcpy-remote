package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"scrcpy-remote/internal/api"
	"scrcpy-remote/internal/devices"
	"scrcpy-remote/internal/tools"
	"scrcpy-remote/web"
)

func openBrowser(url string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	default:
		exec.Command("xdg-open", url).Start()
	}
}

// resolveTool returns name from PATH if present, else the first existing candidate,
// else the bare name (so a clear "not found" error surfaces at use time).
func resolveTool(name string, candidates ...string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return name
}

func main() {
	port := flag.Int("port", devices.DefaultPort, "dashboard port (127.0.0.1)")
	flag.Parse()

	home, _ := os.UserHomeDir()
	cfgPath := filepath.Join(home, ".scrcpy-remote", "config.json")
	bin := filepath.Join(home, ".scrcpy-remote", "bin")

	adbPath := resolveTool("adb", filepath.Join(bin, "adb.exe"), filepath.Join(bin, "adb"))
	scrcpyPath := resolveTool("scrcpy", filepath.Join(bin, "scrcpy.exe"), filepath.Join(bin, "scrcpy"))
	tsPath := resolveTool("tailscale",
		`C:\Program Files\Tailscale\tailscale.exe`,
		`C:\Program Files (x86)\Tailscale\tailscale.exe`,
		"/Applications/Tailscale.app/Contents/MacOS/Tailscale")

	srv := &api.Server{
		ConfigPath:    cfgPath,
		Runner:        tools.ExecRunner{},
		ADBPath:       adbPath,
		ScrcpyPath:    scrcpyPath,
		TailscalePath: tsPath,
		LaunchFn: func(args []string) error {
			cmd := exec.Command(scrcpyPath, args...)
			cmd.Env = append(os.Environ(), "ADB="+adbPath) // scrcpy uses our adb
			return cmd.Start()
		},
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", srv.Handler())
	mux.Handle("/", http.FileServer(http.FS(web.FS())))

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	url := "http://" + addr
	fmt.Println("scrcpy-remote dashboard:", url)
	openBrowser(url)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintln(os.Stderr, "server error:", err)
		os.Exit(1)
	}
}
