package main

import (
	"os"
	"runtime"

	desktopconfig "icoo_claw/desktop/internal/config"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type AppInfo struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	GoVersion     string `json:"goVersion"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	UserConfigDir string `json:"userConfigDir"`
}

type SystemService struct {
	manager *BundledGatewayManager
	store   *desktopconfig.Store
}

func NewSystemService(store *desktopconfig.Store) *SystemService {
	return &SystemService{
		manager: NewBundledGatewayManager(),
		store:   store,
	}
}

func (s *SystemService) GetAppInfo() (*AppInfo, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	return &AppInfo{
		Name:          appName,
		Version:       appVersion,
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
		SetTitle("Choose workspace directory").
		PromptForSingleSelection()
}

func (s *SystemService) ChooseGatewayProgram() (string, error) {
	return application.Get().
		Dialog.
		OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(true).
		SetTitle("Choose gateway program or bundle folder").
		PromptForSingleSelection()
}

func (s *SystemService) ChooseGatewayConfig() (string, error) {
	return application.Get().
		Dialog.
		OpenFile().
		CanChooseDirectories(false).
		CanChooseFiles(true).
		SetTitle("Choose gateway config file").
		PromptForSingleSelection()
}

func (s *SystemService) EnsureBundledGateway(baseURL string) (bool, error) {
	if s.manager == nil {
		s.manager = NewBundledGatewayManager()
	}
	var programPath string
	var configPath string
	if s.store != nil {
		settings, err := s.store.Load()
		if err != nil {
			return false, err
		}
		programPath = settings.Gateway.ProgramPath
		configPath = settings.Gateway.ConfigPath
	}
	return s.manager.EnsureBundledGateway(baseURL, programPath, configPath)
}
