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

func main() {
	port := flag.Int("port", devices.DefaultPort, "dashboard port (127.0.0.1)")
	flag.Parse()

	home, _ := os.UserHomeDir()
	cfgPath := filepath.Join(home, ".scrcpy-remote", "config.json")

	srv := &api.Server{
		ConfigPath: cfgPath,
		Runner:     tools.ExecRunner{},
		ADBPath:       "adb",
		ScrcpyPath:    "scrcpy",
		TailscalePath: "tailscale",
		LaunchFn: func(args []string) error {
			return exec.Command("scrcpy", args...).Start()
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
