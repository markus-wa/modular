package main

import (
	"context"
	"fmt"
	"log"
	"time"

	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
	"go.uber.org/zap"

	"github.com/markus-wa/vlc-sampler/features/input"
)

func run() error {
	ctx := context.Background()

	gamepad, err := input.PollDefault(ctx)
	if err != nil {
		return fmt.Errorf("failed to poll device: %w", err)
	}

	defer gamepad.Close()

	zap.S().Infow("starting", "gamepad", gamepad.Name())

	for event := range gamepad.Poll(ctx) {
		zap.S().Infow("event", "event", event)
	}

	return nil
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("could not initialize logger: %v", err)
	}

	defer logger.Sync()

	zap.ReplaceGlobals(logger)

	t := time.NewTicker(1 * time.Second)

	for range t.C {
		err := run()
		if err != nil {
			zap.S().Errorw("run failed", "error", err)
		}
	}
}
