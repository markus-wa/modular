package input

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/kenshaw/evdev"
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

func PollDefault(ctx context.Context) (*evdev.Evdev, error) {
	devs, err := listDevices()
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}

	var dev *Device

	for _, d := range devs {
		if d.IsGamepad && dev == nil || (dev != nil && !dev.IsCombinedJoyCon() && d.IsCombinedJoyCon()) {
			dev = &d // either the first gamepad or the first combined joycon
		}
	}

	if dev == nil {
		return nil, fmt.Errorf("no gamepad found")
	}

	device, err := evdev.OpenFile(dev.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open device: %w", err)
	}

	return device, nil
}
