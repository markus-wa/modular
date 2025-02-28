package main

import (
	"log"
	"time"

	"github.com/markus-wa/vlc-sampler/features/hud"
)

func main() {
	h, err := hud.NewHud()
	if err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}

	go h.Start()

	for range time.Tick(time.Second) {
		h.SendText("Hello, World!")
	}

	<-make(chan struct{})
}
