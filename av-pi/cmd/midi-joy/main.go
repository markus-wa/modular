package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv"

	"github.com/markus-wa/vlc-sampler/features/input"
	"github.com/markus-wa/vlc-sampler/features/midictl"
)

func run() error {
	for i, port := range midi.GetOutPorts() {
		fmt.Printf("MIDI Port %d: %s\n", i, port.String())
	}

	midiSvc, err := midictl.NewService()
	if err != nil {
		return fmt.Errorf("could not initialize MIDI service: %w", err)
	}
	defer midiSvc.Close()

	midiCtl, err := midictl.NewController(midiSvc, nil)
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
