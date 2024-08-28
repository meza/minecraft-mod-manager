package main

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/meza/minecraft-mod-manager/cmd/mmm"
	"log"
)

func main() {
	err := mmm.Execute()
	if err != nil {
		log.Fatalf("Error executing command: %v", err)
	}
}
