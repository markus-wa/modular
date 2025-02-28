package hud

import (
	_ "embed"
	"log"
	"runtime"
	"slices"
	"time"

	"github.com/go-gl/gl/all-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/nullboundary/glfont"
)

func init() {
	runtime.LockOSThread()
}

type Text struct {
	text      string
	expiresAt time.Time
}

type Hud struct {
	texts []Text
}

//go:embed Roboto-Regular.ttf
var robotoRegularB []byte

func (h *Hud) Start() {
	runtime.LockOSThread()

	err := glfw.Init()
	if err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}

	defer glfw.Terminate()

	mon := glfw.GetPrimaryMonitor()
	mode := mon.GetVideoMode()

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.TransparentFramebuffer, glfw.True)
	glfw.WindowHint(glfw.Floating, glfw.True)
	glfw.WindowHint(glfw.RedBits, mode.RedBits)
	glfw.WindowHint(glfw.GreenBits, mode.GreenBits)
	glfw.WindowHint(glfw.BlueBits, mode.BlueBits)
	glfw.WindowHint(glfw.RefreshRate, mode.RefreshRate)

	window, err := glfw.CreateWindow(mode.Width, mode.Height, "HUD", nil, nil)
	if err != nil {
		log.Panicf("glfw.CreateWindow: %v", err)
	}

	x, _ := window.GetPos()
	window.SetPos(x, 0)
	window.MakeContextCurrent()
	glfw.SwapInterval(1)

	err = gl.Init()
	if err != nil {
		log.Panicf("gl.Init: %v", err)
	}

	font, err := glfont.LoadFontBytes(robotoRegularB, int32(52), mode.Width, mode.Height)
	if err != nil {
		log.Panicf("glfont.LoadFont: %v", err)
	}

	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.ClearColor(0.0, 0.0, 0.0, 0.8)

	for !window.ShouldClose() {
		h.texts = slices.DeleteFunc(h.texts, func(t Text) bool {
			return t.expiresAt.Before(time.Now())
		})

		if len(h.texts) > 0 {
			gl.ClearColor(0.0, 0.0, 0.0, 0.5)
		} else {
			gl.ClearColor(0.0, 0.0, 0.0, 0.0)
		}

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		font.SetColor(1.0, 1.0, 1.0, 0.5)

		for i, t := range h.texts {
			err = font.Printf(100, 100+float32(i*52), 1.0, t.text)
			if err != nil {
				log.Println("font.Printf:", err)
			}
		}

		window.SwapBuffers()
		glfw.PollEvents()
	}
}

func (h *Hud) SendText(txt string) {
	h.texts = append(h.texts, Text{
		text:      txt,
		expiresAt: time.Now().Add(2 * time.Second),
	})
}

func NewHud() (*Hud, error) {
	hud := &Hud{}

	return hud, nil
}
