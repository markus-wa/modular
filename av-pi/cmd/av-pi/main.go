package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	vlc "github.com/adrg/libvlc-go/v3"
	"github.com/kenshaw/evdev"
	"gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
	"go.uber.org/zap"

	"github.com/markus-wa/vlc-sampler/features/cliui"
	"github.com/markus-wa/vlc-sampler/features/hud"
	"github.com/markus-wa/vlc-sampler/features/input"
	"github.com/markus-wa/vlc-sampler/features/midictl"
	"github.com/markus-wa/vlc-sampler/features/sampler"
)

type UI interface {
	SendText(string)
	Start()
}

var uiFlag = flag.String("ui", "cli", "UI to use (cli, hud)")

func run() error {
	err := vlc.Init("--no-autoscale")
	if err != nil {
		return fmt.Errorf("failed to initialize libvlc: %w", err)
	}

	defer vlc.Release()

	var ui UI

	switch *uiFlag {
	case "cli":
		ui = cliui.UI{}

	case "hud":
		ui, err = hud.NewHud()

	default:
		return fmt.Errorf("unknown UI: %s", *uiFlag)
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

	midiCtl, err := midictl.NewController(midiSvc, ui)
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

	samplerCtrl, err := sampler.NewController(smplr, ui)
	if err != nil {
		return fmt.Errorf("could not initialize sampler controller: %w", err)
	}

	go ui.Start()

	ctx := context.Background()

	for range time.Tick(time.Second) {
		err := pollDefaultGamepad(ctx, samplerCtrl, midiCtl, ui)
		if err != nil {
			zap.S().Errorw("pollDefaultGamepad failed", err)
		}
	}

	return nil
}

func pollDefaultGamepad(ctx context.Context, samplerCtrl *sampler.Controller, midiCtl *midictl.Controller, ui UI) error {
	gamepad, err := input.PollDefault(ctx)
	if err != nil {
		return fmt.Errorf("failed to poll device: %w", err)
	}

	defer gamepad.Close()

	fmt.Printf("Device Name: %s\n", gamepad.Name())

	mode := 0
	modeModifier := false

	for event := range gamepad.Poll(ctx) {
		if fmt.Sprint(event.Type) == "Report" {
			continue
		}

		ui.SendText(fmt.Sprintf("%s (%d) %d", event.Type, event.Code, event.Value))

		if modeModifier && event.Type == evdev.BtnMode && event.Value != 0 {
			mode++

			log.Println("mode changed to", mode%2)

			continue
		}

		if event.Type == evdev.BtnZ || event.Type == evdev.AbsoluteZ {
			modeModifier = event.Value != 0
		}

		switch mode % 2 {
		case 0:
			err := samplerCtrl.HandleEvent(event)
			if err != nil {
				log.Println("failed to handle event:", err)
			}

		case 1:
			err := midiCtl.HandleEvent(event)
			if err != nil {
				log.Println("failed to handle event:", err)
			}
		}
	}

	return nil
}

func main() {
	flag.Parse()

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("could not initialize logger: %v", err)
	}

	defer logger.Sync()

	zap.ReplaceGlobals(logger)

	for range time.Tick(1 * time.Second) {
		err := run()
		if err != nil {
			zap.S().Errorw("run failed", "error", err)
		}
	}
}
