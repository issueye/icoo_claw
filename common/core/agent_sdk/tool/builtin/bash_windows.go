//go:build windows

package toolbuiltin

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func bashOutputBaseDir() string {
	return filepath.Join(os.TempDir(), "agentsdk", "bash-output")
}

func newBashCommandContext(ctx context.Context, command string) (*exec.Cmd, string) {
	if bashPath, err := exec.LookPath("bash"); err == nil {
		return exec.CommandContext(ctx, bashPath, "-c", command), "bash"
	}

	command = translateWindowsShellCommand(command)
	if pwshPath, err := exec.LookPath("pwsh"); err == nil {
		return exec.CommandContext(ctx, pwshPath, "-NoProfile", "-NonInteractive", "-Command", command), "pwsh"
	}
	return exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", command), "powershell"
}

func translateWindowsShellCommand(command string) string {
	trimmed := strings.TrimSpace(command)
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return trimmed
	}

	switch strings.ToLower(fields[0]) {
	case "ls":
		return translateWindowsLS(fields)
	case "pwd":
		return "(Get-Location).Path"
	default:
		return trimmed
	}
}

func translateWindowsLS(fields []string) string {
	force := false
	paths := make([]string, 0, 1)
	for _, field := range fields[1:] {
		if strings.HasPrefix(field, "-") {
			if strings.Contains(field, "a") {
				force = true
			}
			continue
		}
		paths = append(paths, field)
	}

	path := "."
	if len(paths) > 0 {
		path = paths[0]
	}

	parts := []string{"Get-ChildItem"}
	if force {
		parts = append(parts, "-Force")
	}
	parts = append(parts, "-LiteralPath", quotePowerShellSingle(path), "|", "Select-Object", "Mode,Length,LastWriteTime,Name", "|", "Format-Table", "-AutoSize")
	return strings.Join(parts, " ")
}

func quotePowerShellSingle(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
