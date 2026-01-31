package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	options, err := loadOptions()
	if err != nil {
		log.Fatal(err)
	}

	server := NewServer(options)
	server.routes()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Wake service listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
