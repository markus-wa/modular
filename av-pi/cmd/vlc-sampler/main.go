package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"time"
	"unicode"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
	vlc "github.com/adrg/libvlc-go/v3"
)

type Mode int

const (
	ModePlaylists Mode = iota
	ModeScreen
	ModeMax
)

type avSampler struct {
	listPlayer      *vlc.ListPlayer
	recorder        *vlc.Player
	screenMediaList *vlc.MediaList

	isRecording      bool
	currentListIndex int
	playlists        []string
	mode             Mode
}

func newAVSampler(playlistDir string) (*avSampler, error) {
	dir, err := os.ReadDir(playlistDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var playlists []string

	for _, entry := range dir {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".xspf") {
			continue
		}

		playlists = append(playlists, path.Join(playlistDir, entry.Name()))
	}

	player, err := vlc.NewListPlayer()
	if err != nil {
		return nil, fmt.Errorf("failed to create listPlayer: %w", err)
	}

	recorder, err := vlc.NewPlayer()
	if err != nil {
		return nil, fmt.Errorf("failed to create recorder: %w", err)
	}

	screenMedia, err := vlc.NewMediaFromScreen(&vlc.MediaScreenOptions{
		FPS: 24,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get media from screen: %w", err)
	}

	screenMediaList, err := vlc.NewMediaList()
	if err != nil {
		return nil, fmt.Errorf("failed to create media list: %w", err)
	}

	err = screenMediaList.AddMedia(screenMedia)
	if err != nil {
		return nil, fmt.Errorf("failed to add media to list: %w", err)
	}

	av := &avSampler{
		listPlayer:      player,
		recorder:        recorder,
		playlists:       playlists,
		screenMediaList: screenMediaList,
	}

	if len(playlists) > 0 {
		err := av.playPlaylist(0)
		if err != nil {
			return nil, fmt.Errorf("failed to play playlist 0: %w", err)
		}
	}

	return av, nil
}

func (s *avSampler) Previous() error {
	err := s.listPlayer.PlayPrevious()
	if err != nil {
		return fmt.Errorf("failed to play previous media: %w", err)
	}

	return nil
}

func (s *avSampler) Next() error {
	err := s.listPlayer.PlayNext()
	if err != nil {
		return fmt.Errorf("failed to play next media: %w", err)
	}

	return nil
}

func (s *avSampler) playPlaylist(i int) error {
	log.Println("starting", s.playlists[i])

	m, err := vlc.NewMediaFromPath(s.playlists[i])
	if err != nil {
		return fmt.Errorf("failed to create media from playlist: %w", err)
	}

	err = m.ParseWithOptions(-1, vlc.MediaFetchLocal|vlc.MediaParseNetwork)
	if err != nil {
		return fmt.Errorf("failed to parse media: %w", err)
	}

	em, err := m.EventManager()
	if err != nil {
		return fmt.Errorf("failed to get event manager: %w", err)
	}

	done := make(chan struct{})

	id, err := em.Attach(vlc.MediaParsedChanged, func(event vlc.Event, i interface{}) {
		close(done)
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to attach event: %w", err)
	}

	defer em.Detach(id)

	select {
	case <-done:

	case <-time.After(5 * time.Second):
		return fmt.Errorf("%w: parsing media took too long", errTimeout)
	}

	ml, err := m.SubItems()
	if err != nil {
		return fmt.Errorf("failed to get media list from playlist media: %w", err)
	}

	err = s.listPlayer.SetPlaybackMode(vlc.Loop)
	if err != nil {
		return fmt.Errorf("failed to set playback mode: %w", err)
	}

	err = s.listPlayer.SetMediaList(ml)
	if err != nil {
		return fmt.Errorf("failed to set media: %w", err)
	}

	if !s.listPlayer.IsPlaying() {
		err = s.listPlayer.Play()
		if err != nil {
			return fmt.Errorf("failed to play media: %w", err)
		}
	} else {
		err = s.listPlayer.PlayAtIndex(0)
		if err != nil {
			return fmt.Errorf("failed to play media: %w", err)
		}
	}

	return nil
}

func (s *avSampler) PreviousPlaylist() error {
	s.currentListIndex--

	if s.currentListIndex < 0 {
		s.currentListIndex = len(s.playlists) - 1
	}

	return s.playPlaylist(s.currentListIndex)
}

func (s *avSampler) NextPlaylist() error {

	s.currentListIndex++

	if s.currentListIndex >= len(s.playlists) {
		s.currentListIndex = 0
	}

	return s.playPlaylist(s.currentListIndex)
}

func (s *avSampler) Close() error {
	err := s.listPlayer.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop listPlayer: %w", err)
	}

	err = s.listPlayer.Release()
	if err != nil {
		return fmt.Errorf("failed to release listPlayer: %w", err)
	}

	return nil
}

var errNotImplemented = errors.New("not implemented")

func (s *avSampler) startRecording(target string) error {
	p, err := s.listPlayer.Player()
	if err != nil {
		return fmt.Errorf("failed to get player: %w", err)
	}

	m, err := p.Media()
	if err != nil {
		return fmt.Errorf("failed to get media: %w", err)
	}

	t, err := m.Type()
	if err != nil {
		return fmt.Errorf("failed to get media type: %w", err)
	}

	// "unknown" can be screen recording
	if t != vlc.MediaTypeStream && t != vlc.MediaTypeUnknown {
		return fmt.Errorf("%w: can't record non-stream media", errNotImplemented)
	}

	recMedia, err := m.Duplicate()
	if err != nil {
		return fmt.Errorf("failed to duplicate media: %w", err)
	}

	err = recMedia.AddOptions(fmt.Sprintf(":sout=#transcode{acodec=mpga, vcodec=h265}:std{access=file,mux=mp4,dst=%s}", target), ":sout-keep")
	if err != nil {
		return fmt.Errorf("failed to add options: %w", err)
	}

	err = s.recorder.SetMedia(recMedia)
	if err != nil {
		return fmt.Errorf("failed to set media: %w", err)
	}

	err = s.recorder.Play()
	if err != nil {
		return fmt.Errorf("failed to play media: %w", err)
	}

	return nil
}

func (s *avSampler) stopRecording() error {
	err := s.recorder.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop recording: %w", err)
	}

	return nil
}

func (s *avSampler) ToggleRecording() error {
	if s.isRecording {
		err := s.stopRecording()
		if err != nil {
			return fmt.Errorf("failed to stop recording: %w", err)
		}

		log.Println("Recording stopped")

		s.isRecording = false

		return nil
	}

	err := s.startRecording(fmt.Sprintf("/tmp/recs/%d.mp4", time.Now().Unix()))
	if err != nil {
		return fmt.Errorf("failed to record media: %w", err)
	}

	log.Println("Recording started")

	s.isRecording = true

	return nil
}

func (s *avSampler) TogglePlayPause() error {
	if s.listPlayer.IsPlaying() {
		err := s.listPlayer.Stop()
		if err != nil {
			return fmt.Errorf("failed to pause: %w", err)
		}
	} else {
		err := s.listPlayer.Play()
		if err != nil {
			return fmt.Errorf("failed to play: %w", err)
		}
	}

	return nil
}

func (s *avSampler) ToggleMode() error {
	s.mode++

	if s.mode >= ModeMax {
		s.mode = 0
	}

	switch s.mode {
	case ModePlaylists:
		err := s.playPlaylist(s.currentListIndex)
		if err != nil {
			return fmt.Errorf("failed to play playlist: %w", err)
		}

	case ModeScreen:
		err := s.listPlayer.SetMediaList(s.screenMediaList)
		if err != nil {
			return fmt.Errorf("failed to set media list: %w", err)
		}

		err = s.listPlayer.PlayAtIndex(0)
		if err != nil {
			return fmt.Errorf("failed to play screen media list: %w", err)
		}

	default:
		panic(fmt.Sprintf("unknown mode %d", s.mode))
	}

	return nil
}

var errTimeout = errors.New("timeout")

func y(ctx context.Context) error {
	err := os.MkdirAll("/tmp/recs", 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := vlc.Init("--quiet"); err != nil {
		return fmt.Errorf("failed to initialize libvlc: %w", err)
	}

	defer vlc.Release()

	av, err := newAVSampler("/home/markus/Playlists")
	if err != nil {
		return fmt.Errorf("failed to create avSampler: %w", err)
	}

	defer av.Close()

	err = keyboard.Listen(func(key keys.Key) (stop bool, err error) {
		if key.Code == keys.CtrlC || key.Code == keys.RuneKey && key.Runes[0] == 'q' {
			return true, nil
		}

		if key.Code == keys.Left {
			err = av.Previous()
			if err != nil {
				return false, fmt.Errorf("failed to play previous media: %w", err)
			}
		}

		if key.Code == keys.Right {
			err = av.Next()
			if err != nil {
				return false, fmt.Errorf("failed to play next media: %w", err)
			}
		}

		if key.Code == keys.ShiftLeft {
			err = av.PreviousPlaylist()
			if err != nil {
				return false, fmt.Errorf("failed to play previous playlist: %w", err)
			}
		}

		if key.Code == keys.ShiftRight {
			err = av.NextPlaylist()
			if err != nil {
				return false, fmt.Errorf("failed to play next playlist: %w", err)
			}
		}

		if key.Code == keys.RuneKey && unicode.ToLower(key.Runes[0]) == 'p' {
			err = av.TogglePlayPause()
			if err != nil {
				return false, fmt.Errorf("failed to play media: %w", err)
			}
		}

		if key.Code == keys.RuneKey && unicode.ToLower(key.Runes[0]) == 'r' {
			err := av.ToggleRecording()
			if err != nil {
				return false, fmt.Errorf("failed to toggle recording: %w", err)
			}

			return false, nil
		}

		if key.Code == keys.RuneKey && unicode.ToLower(key.Runes[0]) == 'm' {
			err := av.ToggleMode()
			if err != nil {
				return false, fmt.Errorf("failed to toggle recording: %w", err)
			}

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
