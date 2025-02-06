package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"time"
	"unicode"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"

	vlc "github.com/adrg/libvlc-go/v3"
)

func playDir(player *vlc.ListPlayer, dir string) error {
	ml, err := vlc.NewMediaList()
	if err != nil {
		return fmt.Errorf("failed to create media list: %w", err)
	}

	d, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// TODO: shuffle dir if desired

	for _, entry := range d {
		if entry.IsDir() {
			continue
		}

		m, err := vlc.NewMediaFromPath(path.Join(dir, entry.Name()))
		if err != nil {
			return fmt.Errorf("failed to create media: %w", err)
		}

		defer m.Release()

		err = ml.AddMedia(m)
		if err != nil {
			return fmt.Errorf("failed to add media: %w", err)
		}
	}

	err = player.SetMediaList(ml)
	if err != nil {
		return fmt.Errorf("failed to set media: %w", err)
	}

	err = player.PlayAtIndex(0)
	if err != nil {
		return fmt.Errorf("failed to play media: %w", err)
	}

	return nil
}

func startRecording(ctx context.Context, recorder *vlc.Player, media *vlc.Media, target string) error {
	recMedia, err := media.Duplicate()
	if err != nil {
		return fmt.Errorf("failed to duplicate media: %w", err)
	}

	err = recMedia.AddOptions(fmt.Sprintf(":sout=#transcode{acodec=mpga, vcodec=h265}:std{access=file,mux=mp4,dst=%s}", target), ":sout-keep")
	if err != nil {
		return fmt.Errorf("failed to add options: %w", err)
	}

	err = recorder.SetMedia(recMedia)
	if err != nil {
		return fmt.Errorf("failed to set media: %w", err)
	}

	err = recorder.Play()
	if err != nil {
		return fmt.Errorf("failed to play media: %w", err)
	}

	return nil
}

func y(ctx context.Context) error {
	err := os.MkdirAll("/tmp/recs", 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := vlc.Init("--quiet"); err != nil {
		return fmt.Errorf("failed to initialize libvlc: %w", err)
	}
	defer vlc.Release()

	player, err := vlc.NewListPlayer()
	if err != nil {
		return fmt.Errorf("failed to create player: %w", err)
	}
	defer func() {
		player.Stop()
		player.Release()
	}()

	err = player.SetPlaybackMode(vlc.Loop)
	if err != nil {
		return fmt.Errorf("failed to set playback mode: %w", err)
	}

	recorder, err := vlc.NewPlayer()
	if err != nil {
		return fmt.Errorf("failed to create recorder: %w", err)
	}
	defer func() {
		recorder.Stop()
		recorder.Release()
	}()

	media, err := vlc.NewMediaFromScreen(nil)
	if err != nil {
		return fmt.Errorf("failed to create media: %w", err)
	}
	defer media.Release()

	ml, err := vlc.NewMediaList()
	if err != nil {
		return fmt.Errorf("failed to create media list: %w", err)
	}

	err = ml.AddMedia(media)
	if err != nil {
		return fmt.Errorf("failed to add media: %w", err)
	}

	err = player.SetMediaList(ml)
	if err != nil {
		return fmt.Errorf("failed to set media: %w", err)
	}

	err = player.Play()
	if err != nil {
		return fmt.Errorf("failed to play media: %w", err)
	}

	isRecording := false

	err = keyboard.Listen(func(key keys.Key) (stop bool, err error) {
		if key.Code == keys.CtrlC || key.Code == keys.RuneKey && key.Runes[0] == 'q' {
			return true, nil
		}

		if key.Code == keys.RuneKey && unicode.ToLower(key.Runes[0]) == 'p' {
			err = playDir(player, "/tmp/recs")
			if err != nil {
				return false, fmt.Errorf("failed to play media: %w", err)
			}
		}

		// start/stop recording
		if key.Code == keys.RuneKey && unicode.ToLower(key.Runes[0]) == 'r' {
			if isRecording {
				err = recorder.Stop()
				if err != nil {
					return false, fmt.Errorf("failed to stop recording: %w", err)
				}

				log.Println("Recording stopped")

				isRecording = false

				return false, nil
			}

			err = startRecording(ctx, recorder, media, fmt.Sprintf("/tmp/recs/%d.mp4", time.Now().Unix()))
			if err != nil {
				return false, fmt.Errorf("failed to record media: %w", err)
			}

			log.Println("Recording started")

			isRecording = true

			return false, nil
		}

		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to listen for keyboard events: %w", err)
	}

	return nil
}

func main() {
	err := y(context.Background())
	if err != nil {
		log.Fatal(err)
	}
}
