package main

import (
	"flag"
	"log"

	"icoo_claw/server/session_store/internal/di"
)

var (
	cfgPath string
)

func main() {
	flag.StringVar(&cfgPath, "config", "runtime/config/session_store.toml", "config file path")
	flag.Parse()

	container, err := di.NewContainer(cfgPath)
	if err != nil {
		log.Fatalf("session store init failed: %v", err)
	}
	defer func() {
		if err := container.Close(); err != nil {
			log.Printf("session store close failed: %v", err)
		}
	}()

	if err := container.Run(); err != nil {
		log.Fatalf("session store stopped: %v", err)
	}
}
