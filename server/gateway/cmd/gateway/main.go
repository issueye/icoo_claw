package main

import (
	"flag"
	"log"

	"icoo_claw/server/gateway/internal/config"
	"icoo_claw/server/gateway/internal/di"
)

var (
	cfgPath string
)

func main() {
	flag.StringVar(&cfgPath, "config", config.DefaultConfigPath, "config file path")
	flag.Parse()

	container, err := di.NewContainer(cfgPath)
	if err != nil {
		log.Fatalf("gateway init failed: %v", err)
	}

	if err := container.Run(); err != nil {
		log.Fatalf("gateway stopped: %v", err)
	}
}
