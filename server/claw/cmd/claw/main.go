package main

import (
	"flag"
	"log"
	"os"

	acp "github.com/coder/acp-go-sdk"
	acpagent "icoo_claw/server/claw/internal/acp"
	"icoo_claw/server/claw/internal/config"
	"icoo_claw/server/claw/internal/di"
)

var (
	cfgPath string
	acpMode bool
)

func main() {
	flag.StringVar(&cfgPath, "config", config.DefaultConfigPath, "config file path")
	flag.BoolVar(&acpMode, "acp", false, "run as ACP stdio agent")
	flag.Parse()

	container, err := di.NewContainer(cfgPath)
	if err != nil {
		log.Fatalf("claw init failed: %v", err)
	}

	if acpMode {
		agent := acpagent.NewAgent(container.Runner)
		conn := acp.NewAgentSideConnection(agent, os.Stdout, os.Stdin)
		agent.SetAgentConnection(conn)
		<-conn.Done()
		return
	}

	if err := container.Run(); err != nil {
		log.Fatalf("claw stopped: %v", err)
	}
}
