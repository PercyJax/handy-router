package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/PercyJax/handy-router/config"
	"github.com/PercyJax/handy-router/handler"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	h := handler.New(cfg)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("handy-router listening on %s", addr)

	if err := http.ListenAndServe(addr, h); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
