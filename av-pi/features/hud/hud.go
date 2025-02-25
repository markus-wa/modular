package hud

import (
	"log"
	"runtime"
	"time"

	"github.com/go-gl/gl/all-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/nullboundary/glfont"
)

const windowWidth = 1920
const windowHeight = 1080

func init() {
	runtime.LockOSThread()
}

type Hud struct {
	text string
}

func (h *Hud) SetText(text string) {
	h.text = text

	if text == "" {
		gl.ClearColor(0.0, 0.0, 0.0, 0.0)
	} else {
		gl.ClearColor(0.0, 0.0, 0.0, 0.5)
	}
}

func NewHud() (*Hud, error) {
	hud := &Hud{}

	go func() {
		if err := glfw.Init(); err != nil {
			log.Fatalln("failed to initialize glfw:", err)
		}
		defer glfw.Terminate()

		glfw.WindowHint(glfw.Resizable, glfw.True)
		glfw.WindowHint(glfw.ContextVersionMajor, 3)
		glfw.WindowHint(glfw.ContextVersionMinor, 2)
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
		glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
		glfw.WindowHint(glfw.TransparentFramebuffer, glfw.True)

		window, _ := glfw.CreateWindow(windowWidth, windowHeight, "glfontExample", nil, nil)

		window.MakeContextCurrent()
		glfw.SwapInterval(1)

		err := gl.Init()
		if err != nil {
			log.Panicf("gl.Init: %v", err)
		}

		//load font (fontfile, font scale, window width, window height
		font, err := glfont.LoadFont("/usr/share/fonts/TTF/Roboto-Regular.ttf", int32(52), windowWidth, windowHeight)
		if err != nil {
			log.Panicf("LoadFont: %v", err)
		}

		gl.Enable(gl.DEPTH_TEST)
		gl.DepthFunc(gl.LESS)
		gl.ClearColor(0.0, 0.0, 0.0, 0.8)

		for !window.ShouldClose() {
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

			font.SetColor(1.0, 1.0, 1.0, 0.5)
			//txt := strings.Clone(hud.text)
			//err = font.Printf(100, 100, 1.0, fmt.Sprint("hey", time.Now().Unix()))
			err = font.Printf(100, 100, 1.0, "bruh")
			if err != nil {
				log.Println("font.Printf:", err)
			}

			window.SwapBuffers()
			glfw.PollEvents()
		}

	}()
	time.Sleep(5 * time.Second)

	return hud, nil
}
