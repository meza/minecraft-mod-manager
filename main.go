package main

import (
	"context"
	"log"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"github.com/meza/minecraft-mod-manager/cmd/mmm"
	"github.com/meza/minecraft-mod-manager/internal/lifecycle"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
)

func main() {
	telemetry.Init()
	handlerID := lifecycle.Register(func(os.Signal) {
		telemetry.Shutdown(context.Background())
	})
	defer lifecycle.Unregister(handlerID)
	defer telemetry.Shutdown(context.Background())

	if err := mmm.Execute(); err != nil {
		log.Printf("Error executing command: %v", err)
		os.Exit(1)
	}
}
