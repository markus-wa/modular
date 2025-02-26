package main

import (
	"context"
	"fmt"
	"log"

	"github.com/kenshaw/evdev"
	"gitlab.com/gomidi/midi/v2/drivers/rtmididrv"

	"github.com/markus-wa/vlc-sampler/features/hud"
	"github.com/markus-wa/vlc-sampler/features/input"
	"github.com/markus-wa/vlc-sampler/features/midi"
	"github.com/markus-wa/vlc-sampler/features/sampler"
)

func run() error {
	theHud, err := hud.NewHud()
	if err != nil {
		return fmt.Errorf("could not initialize HUD: %w", err)
	}

	drv, err := rtmididrv.New()
	if err != nil {
		fmt.Errorf("could not initialize MIDI driver: %w", err)
	}
	defer drv.Close()

	outPorts, err := drv.Outs()
	if err != nil {
		fmt.Errorf("could not get MIDI output ports: %w", err)
	}

	for i, port := range outPorts {
		fmt.Printf("MIDI Port %d: %s\n", i, port.String())
	}

	midiSvc, err := midi.NewService(drv)
	if err != nil {
		return fmt.Errorf("could not initialize MIDI service: %w", err)
	}
	defer midiSvc.Close()

	midiCtl, err := midi.NewController(midiSvc, theHud)
	if err != nil {
		return fmt.Errorf("could not initialize MIDI controller: %w", err)
	}

	smplr, err := sampler.New("/home/markus/Playlists")
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
	err := run()
	if err != nil {
		log.Fatalf("error: %v", err)
	}
}
