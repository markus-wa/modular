package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kenshaw/evdev"
	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	"gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
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

// input -32k to 32k, output 0 to 127 where -32k is 0 and 32k is 127
func step(prev uint8, v int32, max uint8) uint8 {
	step := int8(float64(v) / 32767.0 * float64(max))
	next := int32(prev) + int32(step)

	if next < 0 {
		return 0
	} else if next > 127 {
		return 127
	} else {
		return uint8(next)
	}
}

type gamepad struct {
	midiDrv     *rtmididrv.Driver
	midiPortIdx int
	midiPort    drivers.Out

	x  int32
	y  int32
	rx int32
	ry int32

	lastHat0XCh uint8
	lastHat0YCh uint8

	key0     uint8
	vel0     uint8
	key1     uint8
	vel1     uint8
	lastKey0 uint8
	lastVel0 uint8
	lastKey1 uint8
	lastVel1 uint8

	stepSize uint8

	midiPortModifier bool
}

func (g *gamepad) setStepSize(v uint8) {
	if v < 1 {
		v = 1
	} else if v > 127 {
		v = 127
	}

	g.stepSize = v

	fmt.Println("stepSize:", g.stepSize)
}

func (g *gamepad) incStepSize() {
	inc := g.stepSize / 3

	if inc == 0 {
		inc = 1
	}

	g.setStepSize(g.stepSize + inc)
}

func (g *gamepad) decStepSize() {
	dec := g.stepSize / 4

	if dec == 0 {
		dec = 1
	}

	g.setStepSize(g.stepSize - dec)
}

func (g *gamepad) openMidiPort(i int) error {
	outPorts, err := g.midiDrv.Outs()
	if err != nil {
		return fmt.Errorf("could not get MIDI output ports: %v", err)
	}

	if len(outPorts) == 0 {
		return fmt.Errorf("no MIDI output ports available")
	}

	idx := i % len(outPorts)
	if i < 0 {
		idx = len(outPorts) - 1
	}

	log.Println("changing MIDI port from", g.midiPortIdx, "to", idx)

	newPort := outPorts[idx]

	err = newPort.Open()
	if err != nil {
		return fmt.Errorf("could not open MIDI output port: %v", err)
	}

	if g.midiPort != nil {
		err = g.midiPort.Close()
	}

	g.midiPort = newPort
	g.midiPortIdx = idx

	if err != nil {
		return fmt.Errorf("failed to close old MIDI port: %w", err)
	}

	return nil
}

func (g *gamepad) HandleEvent(event *evdev.EventEnvelope) error {
	if fmt.Sprint(event.Type) == "Report" {
		return nil
	}

	fmt.Println(event.Type, event.Code, event.Value)

	if event.Type == evdev.AbsoluteX {
		g.x = event.Value
	} else if event.Type == evdev.AbsoluteY {
		g.y = event.Value
	} else if event.Type == evdev.AbsoluteRX {
		g.rx = event.Value
	} else if event.Type == evdev.AbsoluteRY {
		g.ry = event.Value
	} else if event.Type == evdev.BtnSelect && event.Value == 1 {
		if g.midiPortModifier {
			err := g.openMidiPort(g.midiPortIdx - 1)
			if err != nil {
				return fmt.Errorf("failed to change MIDI port: %w", err)
			}
		} else {
			g.decStepSize()
		}
	} else if event.Type == evdev.BtnStart && event.Value == 1 {
		if g.midiPortModifier {
			err := g.openMidiPort(g.midiPortIdx + 1)
			if err != nil {
				return fmt.Errorf("failed to change MIDI port: %w", err)
			}
		} else {
			g.incStepSize()
		}
	} else if event.Type == evdev.BtnZ {
		g.midiPortModifier = event.Value == 1
	}

	// gates

	ch, ok := map[any]uint8{
		evdev.BtnX:         5,
		evdev.BtnY:         6,
		evdev.BtnA:         7,
		evdev.BtnB:         8,
		evdev.KeyType(544): 9,
		evdev.KeyType(546): 10,
		evdev.KeyType(547): 11,
		evdev.KeyType(545): 12,
		evdev.BtnTL:        13,
		evdev.BtnTR:        14,
		evdev.BtnTL2:       15,
		evdev.BtnTR2:       16,
	}[event.Type]
	if !ok {
		if event.Type != evdev.AbsoluteHat0Y && event.Type != evdev.AbsoluteHat0X {
			return nil
		}

		if event.Type == evdev.AbsoluteHat0Y {
			if event.Value < 0 {
				ch = 9
			} else if event.Value > 0 {
				ch = 12
			} else {
				ch = g.lastHat0YCh
			}

			g.lastHat0YCh = ch
		}

		if event.Type == evdev.AbsoluteHat0X {
			if event.Value < 0 {
				ch = 10
			} else if event.Value > 0 {
				ch = 11
			} else {
				ch = g.lastHat0XCh
			}

			g.lastHat0XCh = ch
		}
	}

	const (
		gateKey = 60
		gateVel = 100
	)

	msg := midi.NoteOn(ch, gateKey, gateVel)

	if event.Value == 0 {
		msg = midi.NoteOff(ch, gateKey)
	}

	err := g.midiPort.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send MIDI note: %w", err)
	}

	return nil
}

func (g *gamepad) sendMidi() error {
	g.key0 = step(g.key0, g.x, g.stepSize)
	g.vel0 = step(g.vel0, g.y, g.stepSize)
	g.key1 = step(g.key1, g.rx, g.stepSize)
	g.vel1 = step(g.vel1, g.ry, g.stepSize)

	var msgs []midi.Message

	if g.lastKey0 != g.key0 {
		msgs = append(msgs, midi.NoteOff(0, g.lastKey0))
	}

	if g.lastKey0 != g.key0 || g.lastVel0 != g.vel0 {
		msgs = append(msgs, midi.NoteOn(0, g.key0, g.vel0))
	}

	if g.lastKey1 != g.key1 {
		msgs = append(msgs, midi.NoteOff(1, g.lastKey1))
	}

	if g.lastKey1 != g.key1 || g.lastVel1 != g.vel1 {
		msgs = append(msgs, midi.NoteOn(1, g.key1, g.vel1))
	}

	for _, m := range msgs {
		err := g.midiPort.Send(m)
		if err != nil {
			return fmt.Errorf("failed to send MIDI note: %w", err)
		}
	}

	g.lastKey0 = g.key0
	g.lastKey1 = g.key1
	g.lastVel0 = g.vel0
	g.lastVel1 = g.vel1

	return nil
}

func (g *gamepad) Close() error {
	return g.midiPort.Close()
}

func newGamepad(midiDrv *rtmididrv.Driver) (*gamepad, error) {
	pad := &gamepad{
		midiDrv:  midiDrv,
		key0:     63,
		vel0:     63,
		key1:     63,
		vel1:     63,
		stepSize: 8,
	}

	err := pad.openMidiPort(0)
	if err != nil {
		return nil, fmt.Errorf("failed to open MIDI port: %w", err)
	}

	return pad, nil
}

func run() error {
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
		return fmt.Errorf("could not initialize MIDI driver: %v", err)
	}
	defer drv.Close()

	outPorts, err := drv.Outs()
	if err != nil {
		return fmt.Errorf("could not get MIDI output ports: %v", err)
	}

	for i, port := range outPorts {
		fmt.Printf("MIDI Port %d: %s\n", i, port.String())
	}

	pad, err := newGamepad(drv)
	if err != nil {
		return fmt.Errorf("could not initialize gamepad: %v", err)
	}
	defer pad.Close()

	f, err := os.Open(dev.Path)
	if err != nil {
		return fmt.Errorf("failed to open device: %v", err)
	}
	defer f.Close()

	device := evdev.Open(f)
	defer device.Close()

	fmt.Printf("Device Name: %s\n", device.Name())

	ctx := context.Background()

	go func() {
		for {
			time.Sleep(100 * time.Millisecond)

			err := pad.sendMidi()
			if err != nil {
				log.Fatalf("error: %v", err)
			}
		}
	}()

	for event := range device.Poll(ctx) {
		err := pad.HandleEvent(event)
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
