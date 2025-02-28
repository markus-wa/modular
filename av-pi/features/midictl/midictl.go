package midictl

import (
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/kenshaw/evdev"
	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
)

type Service struct {
	portIdx int
	port    drivers.Out
	mu      sync.Mutex

	lastVel0 uint8
	lastVel1 uint8
	lastVel2 uint8
	lastVel3 uint8
}

func NewService() (*Service, error) {
	svc := &Service{}

	idx := slices.IndexFunc(midi.GetOutPorts(), func(o drivers.Out) bool {
		return strings.Contains(o.String(), "CH345")
	})

	err := svc.openMidiPort(idx)
	if err != nil {
		return nil, fmt.Errorf("failed to open MIDI port %d: %w", idx, err)
	}

	return svc, nil
}

func (s *Service) openMidiPort(i int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	outPorts := midi.GetOutPorts()

	if len(outPorts) == 0 {
		return fmt.Errorf("no MIDI output ports available")
	}

	idx := i % len(outPorts)
	if i < 0 {
		idx = len(outPorts) - 1
	}

	log.Println("changing MIDI port from", s.portIdx, "to", idx)

	newPort := outPorts[idx]

	err := newPort.Open()
	if err != nil {
		return fmt.Errorf("could not open MIDI output port: %v", err)
	}

	if s.port != nil {
		err = s.port.Close()
	}

	s.port = newPort
	s.portIdx = idx

	if err != nil {
		return fmt.Errorf("failed to close old MIDI port: %w", err)
	}

	return nil
}

func (s *Service) Send(vel0, vel1, vel2, vel3 uint8) error {
	if vel0 == s.lastVel0 && vel1 == s.lastVel1 && vel2 == s.lastVel2 && vel3 == s.lastVel3 {
		return nil
	}

	var msgs []midi.Message

	log.Println("sending MIDI CCs:", vel0, vel1, vel2, vel3)

	msgs = append(msgs, midi.ControlChange(0, 2, vel0))
	msgs = append(msgs, midi.ControlChange(1, 2, vel1))
	msgs = append(msgs, midi.ControlChange(2, 2, vel2))
	msgs = append(msgs, midi.ControlChange(3, 2, vel3))

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, m := range msgs {
		err := s.port.Send(m)
		if err != nil {
			return fmt.Errorf("failed to send MIDI CCs: %w", err)
		}
	}

	s.lastVel0 = vel0
	s.lastVel1 = vel1
	s.lastVel2 = vel2
	s.lastVel3 = vel3

	return nil
}

func (s *Service) Gate(ch uint8, on bool) error {
	const (
		gateKey = 60
		gateVel = 100
	)

	msg := midi.NoteOn(ch, gateKey, gateVel)

	if !on {
		msg = midi.NoteOff(ch, gateKey)
	}

	log.Println("sending MIDI note:", ch, gateKey, on)

	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.port.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send MIDI note: %w", err)
	}

	return nil
}

func (s *Service) previousPort() error {
	return s.openMidiPort(s.portIdx - 1)
}

func (s *Service) nextPort() error {
	return s.openMidiPort(s.portIdx + 1)
}

func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.port.Close()
}

type UI interface {
	SendText(text string)
}

type Controller struct {
	x  int32
	y  int32
	rx int32
	ry int32

	upOn    bool
	leftOn  bool
	rightOn bool
	downOn  bool
	tlOn    bool
	trOn    bool

	stepSize         uint8
	midiPortModifier bool

	svc *Service
	ui  UI
}

func NewController(svc *Service, ui UI) (*Controller, error) {
	c := &Controller{
		stepSize: 8,
		svc:      svc,
		ui:       ui,
	}

	go c.loop()

	return c, nil
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

func (c *Controller) loop() {
	var (
		key0 uint8 = 63
		vel0 uint8 = 63
		key1 uint8 = 63
		vel1 uint8 = 63
	)

	const stepInterval = 100 * time.Millisecond

	ticker := time.NewTicker(stepInterval)

	for {
		<-ticker.C

		key0 = step(key0, c.x, c.stepSize)
		vel0 = step(vel0, c.y, c.stepSize)
		key1 = step(key1, c.rx, c.stepSize)
		vel1 = step(vel1, c.ry, c.stepSize)

		err := c.svc.Send(key0, vel0, key1, vel1)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
	}
}

func (c *Controller) setStepSize(v uint8) {
	if v < 1 {
		v = 1
	} else if v > 127 {
		v = 127
	}

	c.stepSize = v

	fmt.Println("stepSize:", c.stepSize)
}

func (c *Controller) incStepSize() {
	inc := c.stepSize / 3

	if inc == 0 {
		inc = 1
	}

	c.setStepSize(c.stepSize + inc)
}

func (c *Controller) decStepSize() {
	dec := c.stepSize / 4

	if dec == 0 {
		dec = 1
	}

	c.setStepSize(c.stepSize - dec)
}

func (c *Controller) HandleEvent(event *evdev.EventEnvelope) error {
	if event.Type == evdev.AbsoluteX {
		c.x = event.Value
	} else if event.Type == evdev.AbsoluteY {
		c.y = event.Value
	} else if event.Type == evdev.AbsoluteRX {
		c.rx = event.Value
	} else if event.Type == evdev.AbsoluteRY {
		c.ry = event.Value
	} else if event.Type == evdev.BtnSelect && event.Value == 1 {
		if c.midiPortModifier {
			err := c.svc.previousPort()
			if err != nil {
				return fmt.Errorf("failed to change MIDI port: %w", err)
			}
		} else {
			c.decStepSize()
		}
	} else if event.Type == evdev.BtnStart && event.Value == 1 {
		if c.midiPortModifier {
			err := c.svc.nextPort()
			if err != nil {
				return fmt.Errorf("failed to change MIDI port: %w", err)
			}
		} else {
			c.incStepSize()
		}
	} else if event.Type == evdev.BtnZ {
		c.midiPortModifier = event.Value == 1
	}

	// gates

	ch, ok := map[any]uint8{
		evdev.BtnA:       5,
		evdev.BtnB:       4,
		evdev.BtnX:       7,
		evdev.BtnY:       6,
		evdev.BtnTL2:     14,
		evdev.BtnTR2:     15,
		evdev.AbsoluteZ:  14, // 8bitdo TL2
		evdev.AbsoluteRZ: 15, // 8bitdo TR2
	}[event.Type]

	on := event.Value == 1

	// toggles

	if !ok {
		if event.Value == 0 {
			return nil
		}

		if event.Type == evdev.KeyType(544) { // JoyCon D-Pad
			ch = 8
			c.upOn = !c.upOn
			on = c.upOn
		} else if event.Type == evdev.KeyType(546) {
			ch = 9
			c.leftOn = !c.leftOn
			on = c.leftOn
		} else if event.Type == evdev.KeyType(547) {
			ch = 10
			c.rightOn = !c.rightOn
			on = c.rightOn
		} else if event.Type == evdev.KeyType(545) {
			ch = 11
			c.downOn = !c.downOn
			on = c.downOn
		} else if event.Type == evdev.BtnTL { // triggers
			ch = 12
			c.tlOn = !c.tlOn
			on = c.tlOn
		} else if event.Type == evdev.BtnTR {
			ch = 13
			c.trOn = !c.trOn
			on = c.trOn
		} else if event.Type == evdev.AbsoluteHat0Y { // Switch Pro D-Pad
			if event.Value < 0 {
				ch = 8
				c.upOn = !c.upOn
				on = c.upOn
			} else if event.Value > 0 {
				ch = 11
				c.downOn = !c.downOn
				on = c.downOn
			} else {
				panic("unexpected value")
			}
		} else if event.Type == evdev.AbsoluteHat0X {
			if event.Value < 0 {
				ch = 9
				c.leftOn = !c.leftOn
				on = c.leftOn
			} else if event.Value > 0 {
				ch = 10
				c.rightOn = !c.rightOn
				on = c.rightOn
			} else {
				panic("unexpected value")
			}
		}
	}

	if ch == 0 {
		return nil
	}

	err := c.svc.Gate(ch, on)
	if err != nil {
		return fmt.Errorf("failed to set MIDI gate %d to %t: %w", ch, on, err)
	}

	return nil
}
