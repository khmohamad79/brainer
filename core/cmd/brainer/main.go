package main

import (
	"log"
	"net/http"

	"brainer/internal/api"
	"brainer/internal/config"
	"brainer/internal/fill"
	"brainer/internal/gpt"
	"brainer/internal/jira"
	"brainer/internal/memory"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	store, err := memory.New(cfg.MemoryDir)
	if err != nil {
		log.Fatal(err)
	}
	client := gpt.New(cfg.GPTBaseURL, cfg.GPTToken, cfg.GPTModel)
	srv := &api.Server{
		Store: store,
		Fill: &fill.Runner{
			GPT:   client,
			Store: store,
		},
		Jira: jira.New(cfg.JiraBaseURL, cfg.JiraToken),
	}
	log.Printf("brainer on %s (memory %s, model %s)", cfg.HTTPAddr, cfg.MemoryDir, cfg.GPTModel)
	if err := http.ListenAndServe(cfg.HTTPAddr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}
