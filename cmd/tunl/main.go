package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: tunl <port>")
		fmt.Println("Example: tunl 3000")
		os.Exit(1)
	}

	port, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Printf("Error: '%s' is not a valid port number\n", os.Args[1])
		os.Exit(1)
	}

	if port < 1 || port > 65535 {
		fmt.Printf("Error: Port must be between 1 and 65535\n")
		os.Exit(1)
	}

	fmt.Println("═══════════════════════════════════════")
	fmt.Println("  tunl - expose localhost to the internet")
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("  Target: localhost:%d\n", port)
	fmt.Println("  Status: Starting...")
	fmt.Println("")
	fmt.Println("  [Step 1 Complete - Skeleton Only]")
	fmt.Println("  Real tunnel code coming in Step 3!")
	fmt.Println("═══════════════════════════════════════")
}