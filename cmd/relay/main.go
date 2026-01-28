package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println("═══════════════════════════════════════")
	fmt.Println("  tunl relay server")
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("  Listening on: http://localhost:%s\n", port)
	fmt.Println("  Status: Starting...")
	fmt.Println("")
	fmt.Println("  [Step 1 Complete - Skeleton Only]")
	fmt.Println("  Real relay code coming in Step 3!")
	fmt.Println("═══════════════════════════════════════")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "tunl relay server is running!")
	})

	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}
}