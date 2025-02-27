module github.com/markus-wa/vlc-sampler

go 1.23.4

require (
	github.com/adrg/libvlc-go/v3 v3.1.6
	github.com/eiannone/keyboard v0.0.0-20220611211555-0d226195f203
	github.com/go-gl/gl v0.0.0-20231021071112-07e5d0ea2e71
	github.com/go-gl/glfw/v3.3/glfw v0.0.0-20240506104042-037f3cc74f2a
	github.com/kenshaw/evdev v0.1.0
	github.com/nullboundary/glfont v0.0.0-20230301004353-1696e6150876
	github.com/vladimirvivien/go4vl v0.0.5
	gitlab.com/gomidi/midi/v2 v2.2.19
	go.uber.org/zap v1.27.0
)

require (
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/image v0.3.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
)

replace github.com/nullboundary/glfont => github.com/markus-wa/glfont v0.0.0-20250227203211-173e9444cb33
