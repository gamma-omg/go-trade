package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/gamma-omg/trading-bot/internal/agent"
	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/platform"
)

func main() {
	cfg, err := config.ReadFromFile(os.Getenv("CONFIG"))
	if err != nil {
		log.Fatal(err)
	}

	logger := slog.Default()
	p, err := platform.Create(logger, *cfg)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := agent.NewJsonReportBuilder(logger)
	a := agent.NewTradingAgent(logger, *cfg, p, r)
	a.Run(ctx)
}
