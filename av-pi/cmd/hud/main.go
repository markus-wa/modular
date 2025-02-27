package main

import (
	"log"

	"github.com/markus-wa/vlc-sampler/features/hud"
)

func main() {
	h, err := hud.NewHud()
	if err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}

	h.SetText("Hello, World!")

	<-make(chan struct{})
}
