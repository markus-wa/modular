package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	vlc "github.com/adrg/libvlc-go/v3"
	"github.com/kenshaw/evdev"
	"gitlab.com/gomidi/midi/v2/drivers/rtmididrv"

	"github.com/markus-wa/vlc-sampler/features/midi"
)

type Device struct {
	Name      string
	Path      string
	Serial    string
	IsGamepad bool
}

func (d Device) IsCombinedJoyCon() bool {
	return strings.Contains(d.Name, "Nintendo Switch Combined Joy-Cons")
}

func readDevice(path string) (Device, error) {
	f, err := os.Open(path)
	if err != nil {
		return Device{}, fmt.Errorf("failed to open device: %w", err)
	}
	defer f.Close()

	dev := evdev.Open(f)

	return Device{
		Name:      dev.Name(),
		Path:      path,
		Serial:    dev.Serial(),
		IsGamepad: dev.KeyTypes()[evdev.BtnGamepad] || dev.KeyTypes()[evdev.BtnTrigger] || dev.KeyTypes()[evdev.BtnSelect],
	}, nil
}

func listDevices() ([]Device, error) {
	dir, err := os.ReadDir("/dev/input")
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	devices := make([]Device, 0, len(dir))

	for _, entry := range dir {
		if !strings.HasPrefix(entry.Name(), "event") {
			continue
		}

		dev, err := readDevice("/dev/input/" + entry.Name())
		if err != nil {
			if errors.Is(err, os.ErrPermission) {
				continue
			}

			return nil, fmt.Errorf("failed to read device: %w", err)
		}

		if !dev.IsGamepad {
			continue
		}

		devices = append(devices, dev)
	}

	return devices, nil
}

func run() error {
	err := os.MkdirAll("/tmp/recs", 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := vlc.Init("--quiet"); err != nil {
		return fmt.Errorf("failed to initialize libvlc: %w", err)
	}

	defer vlc.Release()

	devs, err := listDevices()
	if err != nil {
		return fmt.Errorf("failed to list devices: %w", err)
	}

	var dev *Device

	for _, d := range devs {
		if d.IsGamepad && dev == nil || (dev != nil && !dev.IsCombinedJoyCon() && d.IsCombinedJoyCon()) {
			dev = &d // either the first gamepad or the first combined joycon
		}
	}

	if dev == nil {
		return fmt.Errorf("no gamepad found")
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

	f, err := os.Open(dev.Path)
	if err != nil {
		return fmt.Errorf("failed to open device: %w", err)
	}
	defer f.Close()

	device := evdev.Open(f)
	defer device.Close()

	fmt.Printf("Device Name: %s\n", device.Name())

	ctx := context.Background()

	for event := range device.Poll(ctx) {
		err := midiCtl.HandleEvent(event)
		if err != nil {
			log.Println("failed to handle event:", err)
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
