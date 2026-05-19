package main

import (
	"flag"
	"log"

	"icoo_claw/server/claw/internal/di"
)

var (
	cfgPath string
)

func main() {
	flag.StringVar(&cfgPath, "config", "runtime/config/claw.toml", "config file path")
	flag.Parse()

	container, err := di.NewContainer(cfgPath)
	if err != nil {
		log.Fatalf("claw init failed: %v", err)
	}

	if err := container.Run(); err != nil {
		log.Fatalf("claw stopped: %v", err)
	}
}
