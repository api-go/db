package db

import (
	"os/exec"
	"strings"
)

func runShell(command string) string {
	cmd := exec.Command("bash", "-c", command)
	out, _ := cmd.CombinedOutput()
	return strings.TrimSpace(string(out))
}
