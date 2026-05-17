package main

import (
	"log"

	"icoo_claw/server/claw/internal/di"
)

func main() {
	container, err := di.NewContainer()
	if err != nil {
		log.Fatalf("claw init failed: %v", err)
	}

	if err := container.Run(); err != nil {
		log.Fatalf("claw stopped: %v", err)
	}
}
