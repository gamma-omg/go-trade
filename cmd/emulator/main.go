package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/gamma-omg/trading-bot/internal/agent"
	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/platform/emulator"
)

func main() {
	cfg, err := config.ReadFromFile(os.Getenv("CONFIG"))
	if err != nil {
		log.Fatal(err)
	}

	cfgEmu, ok := cfg.PlatformRef.Platform.(config.Emulator)
	if !ok {
		log.Fatal("unsupported platform")
	}

	logger := slog.Default()

	emu, err := emulator.NewTradingEmulator(logger, cfgEmu)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		err := emu.Run(ctx)
		if err != nil {
			log.Fatal(err)
		}
	}()

	a := agent.NewTradingAgent(logger, *cfg, emu, &emu.PosMan, emu.Acc)
	a.Run(context.Background())
	<-done
}
