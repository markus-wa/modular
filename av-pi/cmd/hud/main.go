package main

import (
	"log"
	"runtime"

	"github.com/go-gl/gl/all-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/nullboundary/glfont"
)

const windowWidth = 1920
const windowHeight = 1080

func init() {
	runtime.LockOSThread()
}

func main() {

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

	if err := gl.Init(); err != nil {
		panic(err)
	}

	//load font (fontfile, font scale, window width, window height
	font, err := glfont.LoadFont("/usr/share/fonts/TTF/Roboto-Regular.ttf", int32(52), windowWidth, windowHeight)
	if err != nil {
		log.Panicf("LoadFont: %v", err)
	}

	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.ClearColor(0.0, 0.0, 0.0, 0.0)

	for !window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		//set color and draw text

		font.SetColor(1.0, 1.0, 1.0, 0.5) //r,g,b,a font color
		//font.Printf(100, 100, 1.0, "Lorem ipsum dolor sit amet, consectetur adipiscing elit.") //x,y,scale,string,printf args
		font.Printf(100, 100, 1.0, "") //x,y,scale,string,printf args

		window.SwapBuffers()
		glfw.PollEvents()

	}
}
