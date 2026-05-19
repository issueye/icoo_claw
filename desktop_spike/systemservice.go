package main

import (
	"os"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type AppInfo struct {
	Name          string `json:"name"`
	GoVersion     string `json:"goVersion"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	UserConfigDir string `json:"userConfigDir"`
}

type SystemService struct{}

func (s *SystemService) GetAppInfo() (*AppInfo, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	return &AppInfo{
		Name:          "desktop_spike",
		GoVersion:     runtime.Version(),
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		UserConfigDir: configDir,
	}, nil
}

func (s *SystemService) ChooseDirectory() (string, error) {
	return application.Get().
		Dialog.
		OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		SetTitle("Choose a directory for desktop spike").
		PromptForSingleSelection()
}
