package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/kenshaw/evdev"
	"gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
	"go.uber.org/zap"

	"github.com/markus-wa/vlc-sampler/features/hud"
	"github.com/markus-wa/vlc-sampler/features/input"
	"github.com/markus-wa/vlc-sampler/features/midictl"
	"github.com/markus-wa/vlc-sampler/features/sampler"
)

func run() error {
	theHud, err := hud.NewHud()
	if err != nil {
		return fmt.Errorf("could not initialize HUD: %w", err)
	}

	drv, err := rtmididrv.New()
	if err != nil {
		return fmt.Errorf("could not initialize MIDI driver: %w", err)
	}
	defer drv.Close()

	outPorts, err := drv.Outs()
	if err != nil {
		return fmt.Errorf("could not get MIDI output ports: %w", err)
	}

	for i, port := range outPorts {
		zap.S().Infow("MIDI Port", "index", i, "name", port.String())
	}

	midiSvc, err := midictl.NewService()
	if err != nil {
		return fmt.Errorf("could not initialize MIDI service: %w", err)
	}
	defer midiSvc.Close()

	midiCtl, err := midictl.NewController(midiSvc, theHud)
	if err != nil {
		return fmt.Errorf("could not initialize MIDI controller: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get user $HOME dir: %w", err)
	}

	smplr, err := sampler.New(path.Join(home, "Playlists"))
	if err != nil {
		return fmt.Errorf("could not initialize sampler: %w", err)
	}

	samplerCtrl, err := sampler.NewController(smplr, theHud)
	if err != nil {
		return fmt.Errorf("could not initialize sampler controller: %w", err)
	}

	ctx := context.Background()

	mode := 0

	gamepad, err := input.PollDefault(ctx)
	if err != nil {
		return fmt.Errorf("failed to poll device: %w", err)
	}

	defer gamepad.Close()

	fmt.Printf("Device Name: %s\n", gamepad.Name())

	for event := range gamepad.Poll(ctx) {
		if event.Type == evdev.KeyMode || event.Type == evdev.BtnMode {
			mode++

			continue
		}

		switch mode % 2 {
		case 0:
			err := midiCtl.HandleEvent(event)
			if err != nil {
				log.Println("failed to handle event:", err)
			}

		case 1:
			err := samplerCtrl.HandleEvent(event)
			if err != nil {
				log.Println("failed to handle event:", err)
			}
		}
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
