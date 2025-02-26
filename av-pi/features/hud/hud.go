package hud

import (
	"log"
	"runtime"

	"github.com/go-gl/gl/all-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/nullboundary/glfont"
)

func init() {
	runtime.LockOSThread()
}

type Hud struct {
	text string
}

func (h *Hud) SetText(text string) {
	h.text = text
}

func NewHud() (*Hud, error) {
	hud := &Hud{}

	go func() {
		runtime.LockOSThread()

		if err := glfw.Init(); err != nil {
			log.Fatalln("failed to initialize glfw:", err)
		}
		defer glfw.Terminate()

		glfw.WindowHint(glfw.Resizable, glfw.False)
		glfw.WindowHint(glfw.ContextVersionMajor, 3)
		glfw.WindowHint(glfw.ContextVersionMinor, 3)
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
		glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
		glfw.WindowHint(glfw.TransparentFramebuffer, glfw.True)
		glfw.WindowHint(glfw.Floating, glfw.True)

		_, _, w, h := glfw.GetPrimaryMonitor().GetWorkarea()

		window, err := glfw.CreateWindow(w, h, "glfontExample", nil, nil)
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

		//load font (fontfile, font scale, window width, window height
		font, err := glfont.LoadFont("/usr/share/fonts/TTF/Roboto-Regular.ttf", int32(52), w, h)
		if err != nil {
			log.Panicf("glfont.LoadFont: %v", err)
		}

		gl.Enable(gl.DEPTH_TEST)
		gl.DepthFunc(gl.LESS)
		gl.ClearColor(0.0, 0.0, 0.0, 0.8)

		for !window.ShouldClose() {
			if hud.text == "" {
				gl.ClearColor(0.0, 0.0, 0.0, 0.0)
			} else {
				gl.ClearColor(0.0, 0.0, 0.0, 0.5)
			}

			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

			font.SetColor(1.0, 1.0, 1.0, 0.5)
			err = font.Printf(100, 100, 1.0, hud.text)
			if err != nil {
				log.Println("font.Printf:", err)
			}

			window.SwapBuffers()
			glfw.PollEvents()
		}
	}()

	return hud, nil
}
