//go:build windows

package toolbuiltin

import (
	"context"
	"strings"
	"testing"
)

func TestTranslateWindowsLS(t *testing.T) {
	command := translateWindowsShellCommand("ls -la")
	if !strings.Contains(command, "Get-ChildItem") {
		t.Fatalf("command = %q, want Get-ChildItem", command)
	}
	if !strings.Contains(command, "-Force") {
		t.Fatalf("command = %q, want -Force", command)
	}
}

func TestBashToolRunsListCommandWithoutBashOnWindows(t *testing.T) {
	tool := NewBashToolWithRoot(t.TempDir())
	tool.AllowShellMetachars(true)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "ls -la",
	})
	if err != nil {
		t.Fatalf("execute ls -la: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("result = %+v, want success", result)
	}
	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("result data = %T, want map", result.Data)
	}
	if data["shell"] == "" {
		t.Fatalf("result data = %+v, want shell metadata", data)
	}
}
