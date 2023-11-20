package main

import (
	"log"

	"github.com/gcleroux/projet-ift605/pkg/server"
)

func main() {
	srv := server.NewHTTPServer(":8080")
	log.Fatal(srv.ListenAndServe())
}
