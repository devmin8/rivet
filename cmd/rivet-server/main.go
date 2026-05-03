package main

import (
	"log"

	"github.com/devmin8/rivet/internal/server"
)

func main() {
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
