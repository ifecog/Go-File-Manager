package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	r := SetupRouter()

	port := ":8080"
	fmt.Printf("🚀 CLRD Server running on http://localhost%s\n", port)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
