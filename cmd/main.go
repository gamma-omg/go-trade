package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/gamma-omg/trading-bot/internal/agent"
	"github.com/gamma-omg/trading-bot/internal/config"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.ReadFromFile(os.Getenv("CONFIG"))
	if err != nil {
		log.Fatal(err)
	}

	logger := slog.Default()

	r := agent.NewJsonReportBuilder(logger)
	a, err := agent.NewTradingAgent(logger, *cfg, r)
	if err != nil {
		log.Fatal(err)
	}

	a.Run(ctx)
}
