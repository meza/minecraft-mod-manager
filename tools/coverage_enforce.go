package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "total:") {
			fields := strings.Fields(line)
			if len(fields) < 3 {
				fmt.Fprintln(os.Stderr, "Malformed total line in go tool cover output")
				os.Exit(2)
			}
			coverage := fields[2]
			if coverage != "100.0%" {
				fmt.Printf("Coverage is not 100%%: %s\n", coverage)
				os.Exit(1)
			}
			fmt.Println("Coverage is 100%")
			found = true
			break
		}
	}
	if !found {
		fmt.Fprintln(os.Stderr, "No total line found in go tool cover output")
		os.Exit(2)
	}
}
