package main

import (
	"log"

	"icoo_claw/server/session_store/internal/di"
)

func main() {
	container, err := di.NewContainer()
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
