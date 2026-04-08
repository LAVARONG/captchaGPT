package main

import (
	"log"

	"captchagpt/internal/config"
	"captchagpt/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("build server: %v", err)
	}

	log.Printf("captcha api listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
