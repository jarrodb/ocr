package main

import (
	"log"

	"github.com/jarrodb/ocr/pkg/config"
	"github.com/jarrodb/ocr/pkg/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
}
