package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/PercyJax/handy-router/config"
	"github.com/PercyJax/handy-router/handler"
	"github.com/PercyJax/handy-router/indicator"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	h := handler.New(cfg)

	var ind *indicator.Indicator
	if cfg.Indicator.Enabled {
		ind = indicator.New()
		if err := ind.Start(); err != nil {
			log.Printf("Warning: Failed to start indicator: %v", err)
		} else {
			log.Printf("Indicator enabled (D-Bus notifications)")
		}
	}

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("handy-router listening on %s", addr)

	if err := http.ListenAndServe(addr, h); err != nil {
		log.Fatalf("Server failed: %v", err)
	}

	if ind != nil {
		ind.Stop()
	}
}
