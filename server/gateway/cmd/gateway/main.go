package main

import (
	"log"

	"icoo_claw/server/gateway/internal/di"
)

func main() {
	container, err := di.NewContainer()
	if err != nil {
		log.Fatalf("gateway init failed: %v", err)
	}

	if err := container.Run(); err != nil {
		log.Fatalf("gateway stopped: %v", err)
	}
}
