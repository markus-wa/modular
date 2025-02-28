package sampler

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"time"

	vlc "github.com/adrg/libvlc-go/v3"
	"github.com/kenshaw/evdev"
	"github.com/vladimirvivien/go4vl/device"
	"go.uber.org/zap"
)

type Mode int

const (
	ModeStream Mode = iota
	ModePlaylists
	ModeMax
)

type Sampler struct {
	listPlayer      *vlc.ListPlayer
	recorder        *vlc.Player
	streamMediaList *vlc.MediaList

	isRecording      bool
	currentListIndex int
	playlists        []string
	mode             Mode
}

func New(playlistDir string) (*Sampler, error) {
	err := os.MkdirAll(playlistDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	err = os.MkdirAll("/tmp/recs", 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

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

	listPlayer, err := vlc.NewListPlayer()
	if err != nil {
		return nil, fmt.Errorf("failed to create listPlayer: %w", err)
	}

	err = listPlayer.SetPlaybackMode(vlc.Loop)
	if err != nil {
		return nil, fmt.Errorf("failed to set playback mode: %w", err)
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

	err = screenMedia.AddOptions(":live-caching=50")
	if err != nil {
		return nil, fmt.Errorf("failed to add options: %w", err)
	}

	streamMediaList, err := vlc.NewMediaList()
	if err != nil {
		return nil, fmt.Errorf("failed to create media list: %w", err)
	}

	err = streamMediaList.AddMedia(screenMedia)
	if err != nil {
		return nil, fmt.Errorf("failed to add media to list: %w", err)
	}

	zap.S().Infow("added stream screen", "index", 0)

	devs, err := os.ReadDir("/dev")
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	i := 1

	for _, dev := range devs {
		if !strings.HasPrefix(dev.Name(), "video") {
			continue
		}

		path := "/dev/" + dev.Name()

		dev, err := device.Open(path, device.WithBufferSize(1))
		if err != nil {
			zap.S().Errorw("failed to open device", "path", path, err)

			continue
		}

		const deviceCapabilityVideoCapture = 1 << 0

		if dev.Capability().DeviceCapabilities&deviceCapabilityVideoCapture == 0 {
			continue // not a video capture device
		}

		url := "v4l2://" + path

		devMedia, err := vlc.NewMediaFromURL(url)
		if err != nil {
			return nil, fmt.Errorf("failed to get media from screen: %w", err)
		}

		err = devMedia.AddOptions(":live-caching=50")
		if err != nil {
			return nil, fmt.Errorf("failed to add options: %w", err)
		}

		err = streamMediaList.AddMedia(devMedia)
		if err != nil {
			return nil, fmt.Errorf("failed to add media to list: %w", err)
		}

		zap.S().Infow("added stream device", "path", path, "url", url, "index", i)

		i++
	}

	av := &Sampler{
		listPlayer:      listPlayer,
		recorder:        recorder,
		playlists:       playlists,
		streamMediaList: streamMediaList,
	}

	err = av.playStreamList()
	if err != nil {
		return nil, fmt.Errorf("failed to play stream list: %w", err)
	}

	return av, nil
}

func (s *Sampler) Previous() error {
	pl, err := s.listPlayer.Player()
	if err != nil {
		return fmt.Errorf("failed to get player: %w", err)
	}

	m, err := pl.Media()
	if err != nil {
		return fmt.Errorf("failed to get media: %w", err)
	}

	ml := s.listPlayer.MediaList()

	i, err := ml.IndexOfMedia(m)
	if err != nil {
		return fmt.Errorf("failed to get index of media: %w", err)
	}

	n, err := ml.Count()
	if err != nil {
		return fmt.Errorf("failed to get media list count: %w", err)
	}

	if i == 0 {
		err := s.listPlayer.PlayAtIndex(uint(n - 1))
		if err != nil {
			return fmt.Errorf("failed to play last media: %w", err)
		}

		return nil
	}

	err = s.listPlayer.PlayPrevious()
	if err != nil {
		return fmt.Errorf("failed to play previous media: %w", err)
	}

	return nil
}

func (s *Sampler) Next() error {
	pl, err := s.listPlayer.Player()
	if err != nil {
		return fmt.Errorf("failed to get player: %w", err)
	}

	m, err := pl.Media()
	if err != nil {
		return fmt.Errorf("failed to get media: %w", err)
	}

	ml := s.listPlayer.MediaList()

	i, err := ml.IndexOfMedia(m)
	if err != nil {
		return fmt.Errorf("failed to get index of media: %w", err)
	}

	n, err := ml.Count()
	if err != nil {
		return fmt.Errorf("failed to get media list count: %w", err)
	}

	if i == n-1 {
		err := s.listPlayer.PlayAtIndex(0)
		if err != nil {
			return fmt.Errorf("failed to play first media: %w", err)
		}

		return nil
	}

	err = s.listPlayer.PlayNext()
	if err != nil {
		return fmt.Errorf("failed to play next media: %w", err)
	}

	return nil
}

func (s *Sampler) playStreamList() error {
	err := s.listPlayer.SetMediaList(s.streamMediaList)
	if err != nil {
		return fmt.Errorf("failed to set media list: %w", err)
	}

	n, err := s.streamMediaList.Count()
	if err != nil {
		return fmt.Errorf("failed to get media list count: %w", err)
	}

	for i := 0; i < n; i++ {
		err = s.listPlayer.PlayAtIndex(uint(i))
		if err != nil {
			zap.S().Errorw("failed to play media", "index", i, err)

			continue
		}

		pl, err := s.listPlayer.Player()
		if err != nil {
			return fmt.Errorf("failed to get player: %w", err)
		}

		err = pl.SetScale(0)
		if err != nil {
			return fmt.Errorf("failed to set scale: %w", err)
		}

		const full = true

		if full {
			err = pl.SetFullScreen(true)
			if err != nil {
				return fmt.Errorf("failed to set fullscreen: %w", err)
			}
		}

		break
	}

	return nil
}

func (s *Sampler) playPlaylist(i int) error {
	if i >= len(s.playlists) {
		return fmt.Errorf("playlist %d doesn't exist", i)
	}

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

	err = s.listPlayer.SetMediaList(ml)
	if err != nil {
		return fmt.Errorf("failed to set media: %w", err)
	}

	if !s.listPlayer.IsPlaying() {
		err = s.listPlayer.Play()
		if err != nil {
			return fmt.Errorf("failed to play playlist %q: %w", s.playlists[i], err)
		}
	} else {
		err = s.listPlayer.PlayAtIndex(0)
		if err != nil {
			return fmt.Errorf("failed to play media 0 in %q: %w", s.playlists[i], err)
		}
	}

	return nil
}

func (s *Sampler) PreviousPlaylist() error {
	s.currentListIndex--

	if s.currentListIndex < 0 {
		s.currentListIndex = len(s.playlists) - 1
	}

	return s.playPlaylist(s.currentListIndex)
}

func (s *Sampler) NextPlaylist() error {

	s.currentListIndex++

	if s.currentListIndex >= len(s.playlists) {
		s.currentListIndex = 0
	}

	return s.playPlaylist(s.currentListIndex)
}

func (s *Sampler) Close() error {
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

func (s *Sampler) startRecording(target string) error {
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

func (s *Sampler) stopRecording() error {
	err := s.recorder.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop recording: %w", err)
	}

	return nil
}

func (s *Sampler) ToggleRecording() error {
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

func (s *Sampler) TogglePlayPause() error {
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

func (s *Sampler) ToggleMode() error {
	s.mode++

	if s.mode >= ModeMax {
		s.mode = 0
	}

	switch s.mode {
	case ModeStream:
		err := s.playStreamList()
		if err != nil {
			return fmt.Errorf("failed to play stream list: %w", err)
		}

	case ModePlaylists:
		err := s.playPlaylist(s.currentListIndex)
		if err != nil {
			return fmt.Errorf("failed to play playlist: %w", err)
		}

	default:
		panic(fmt.Sprintf("unknown mode %d", s.mode))
	}

	return nil
}

var errTimeout = errors.New("timeout")

type UI interface {
	SendText(string)
}

type Controller struct {
	sampler *Sampler
	ui      UI

	playlistModifier bool
}

func NewController(svc *Sampler, ui UI) (*Controller, error) {
	c := &Controller{
		sampler: svc,
		ui:      ui,
	}

	return c, nil
}

func (c *Controller) HandleEvent(event *evdev.EventEnvelope) error {
	if fmt.Sprint(event.Type) == "Report" {
		return nil
	}

	if event.Type == evdev.BtnSelect && event.Value == 1 {
		if c.playlistModifier {
			err := c.sampler.PreviousPlaylist()
			if err != nil {
				return fmt.Errorf("failed to play previous playlist: %w", err)
			}
		} else {
			err := c.sampler.Previous()
			if err != nil {
				return fmt.Errorf("failed to play previous media %w", err)
			}
		}
	} else if event.Type == evdev.BtnStart && event.Value == 1 {
		if c.playlistModifier {
			err := c.sampler.NextPlaylist()
			if err != nil {
				return fmt.Errorf("failed to play next playlist: %w", err)
			}
		} else {
			err := c.sampler.Next()
			if err != nil {
				return fmt.Errorf("failed to play next media: %w", err)
			}
		}
	} else if event.Type == evdev.BtnStart && event.Value == 1 {
		err := c.sampler.TogglePlayPause()
		if err != nil {
			return fmt.Errorf("failed to toggle play/pause: %w", err)
		}
	} else if event.Type == evdev.BtnSelect && event.Value == 1 {
		err := c.sampler.ToggleRecording()
		if err != nil {
			return fmt.Errorf("failed to toggle recording: %w", err)
		}
	} else if event.Type == evdev.BtnZ {
		c.playlistModifier = event.Value == 1
	} else if event.Type == evdev.BtnMode && event.Value == 1 {
		err := c.sampler.ToggleMode()
		if err != nil {
			return fmt.Errorf("failed to toggle mode: %w", err)
		}
	}

	return nil
}
