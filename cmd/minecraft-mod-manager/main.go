package main

import (
	_ "github.com/joho/godotenv/autoload"
	"os"
)
import (
	"fmt"
)

func main() {
	fmt.Printf(os.Getenv("MODRINTH_API_KEY"))
}
