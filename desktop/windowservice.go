package main

import "github.com/wailsapp/wails/v3/pkg/application"

const acpMonitorWindowName = "acp-monitor"

type WindowService struct {
	app *application.App
}

func NewWindowService() *WindowService {
	return &WindowService{}
}

func (s *WindowService) SetApp(app *application.App) {
	s.app = app
}

func (s *WindowService) OpenACPMonitorWindow() {
	if s == nil || s.app == nil {
		return
	}

	if window, ok := s.app.Window.GetByName(acpMonitorWindowName); ok {
		window.Show()
		window.Focus()
		return
	}

	s.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             acpMonitorWindowName,
		Title:            "ACP 事件监控",
		Width:            1120,
		Height:           760,
		MinWidth:         880,
		MinHeight:        560,
		URL:              "/#/acp-monitor",
		BackgroundColour: application.NewRGB(9, 12, 19),
	})
}
