package cliui

import "fmt"

type UI struct {
}

func (ui UI) Start() {
	// NOOP
}

func (ui UI) SendText(text string) {
	fmt.Println("cliui:", text)
}
