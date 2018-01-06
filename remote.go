package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)


func remote() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Away is just AWAY!")
	})

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
