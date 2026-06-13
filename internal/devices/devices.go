package devices

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const DefaultPort = 8787

type Device struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IP       string `json:"ip"`
	ADBPort  int    `json:"adbPort"`
	PairPort int    `json:"pairPort"`
}

type Config struct {
	Port    int      `json:"port"`
	Devices []Device `json:"devices"`
}

// Load reads config from path. A missing file returns a default config (not an error).
func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{Port: DefaultPort}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, err
	}
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	return c, nil
}

// Save writes config atomically (temp file + rename).
func Save(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// AddOrUpdate replaces a device with the same ID, or appends it. Returns a new Config.
func (c Config) AddOrUpdate(d Device) Config {
	out := Config{Port: c.Port}
	replaced := false
	for _, e := range c.Devices {
		if e.ID == d.ID {
			out.Devices = append(out.Devices, d)
			replaced = true
		} else {
			out.Devices = append(out.Devices, e)
		}
	}
	if !replaced {
		out.Devices = append(out.Devices, d)
	}
	return out
}

// Delete removes the device with the given ID. Returns a new Config.
func (c Config) Delete(id string) Config {
	out := Config{Port: c.Port}
	for _, e := range c.Devices {
		if e.ID != id {
			out.Devices = append(out.Devices, e)
		}
	}
	return out
}
