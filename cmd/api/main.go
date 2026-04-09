package main

import (
	"context"
	"log"
	"time"

	"captchagpt/internal/config"
	"captchagpt/internal/server"
	"captchagpt/internal/upstream"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if cfg.StartupSelfTest {
		runStartupSelfTest(cfg)
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

func runStartupSelfTest(cfg config.Config) {
	client, err := upstream.NewVisionClient(cfg)
	if err != nil {
		log.Printf("startup self-test skipped: build upstream client failed: %v", err)
		return
	}

	log.Printf("startup self-test begin: thinking=%t timeout_seconds=%d model=%s", cfg.EnableThinking, cfg.SelfTestTimeoutS, cfg.ModelName)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.SelfTestTimeoutS)*time.Second)
	defer cancel()

	result, err := client.SelfTest(ctx, cfg.ModelName)
	if err != nil {
		log.Printf("startup self-test failed: thinking=%t status=%d duration_ms=%d error=%v", cfg.EnableThinking, result.StatusCode, result.DurationMS, err)
		return
	}

	log.Printf("startup self-test ok: thinking=%t status=%d duration_ms=%d reply=%q", cfg.EnableThinking, result.StatusCode, result.DurationMS, result.Reply)
}
