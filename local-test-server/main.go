package main

import (
	"fmt"
	"log"
	"net/http"
)

const (
	APP_PORT = ":8080"
)

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Wow!, you managed to access the server, Congrats!!!")
	})
	log.Println("Starting server on port: ", APP_PORT)
	if err := http.ListenAndServe(APP_PORT, nil); err != nil {
		log.Fatal("failed to start server on port: ", APP_PORT)
	}
}
