package tools

import (
	"os/exec"
	"strings"
)

// Runner runs an external command and returns combined stdout/stderr as a string.
type Runner interface {
	Run(name string, args ...string) (string, error)
}

// ExecRunner is the production Runner backed by os/exec.
type ExecRunner struct{}

func (ExecRunner) Run(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
