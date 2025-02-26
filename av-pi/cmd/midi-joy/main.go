package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"gitlab.com/gomidi/midi/v2/drivers/rtmididrv"

	"github.com/markus-wa/vlc-sampler/features/input"
	"github.com/markus-wa/vlc-sampler/features/midi"
)

func run() error {
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

	midiCtl, err := midi.NewController(midiSvc, nil)
	if err != nil {
		return fmt.Errorf("could not initialize MIDI controller: %w", err)
	}

	ctx := context.Background()

	gamepad, err := input.PollDefault(ctx)
	if err != nil {
		return fmt.Errorf("failed to poll device: %w", err)
	}

	defer gamepad.Close()

	fmt.Printf("Device Name: %s\n", gamepad.Name())

	for event := range gamepad.Poll(ctx) {
		err := midiCtl.HandleEvent(event)
		if err != nil {
			log.Println("failed to handle event:", err)
		}
	}

	return nil
}

func main() {
	t := time.NewTicker(1 * time.Second)

	for range t.C {
		err := run()
		if err != nil {
			log.Println("error:", err)
		}
	}
}
