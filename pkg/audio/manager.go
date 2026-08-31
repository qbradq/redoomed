package audio

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"

	"github.com/qbradq/redoomed/pkg/wad"
)

// MusicManager manages background music playback, track transitions, volume, and looping.
type MusicManager struct {
	mu           sync.Mutex
	audioCtx     *audio.Context
	wadFile      *wad.WAD
	player       *audio.Player
	currentTrack string
	volume       float64
	muted        bool
	sampleRate   int
}

// NewMusicManager initializes or reuses the audio context and creates a new MusicManager instance.
func NewMusicManager(w *wad.WAD) *MusicManager {
	ctx := audio.CurrentContext()
	if ctx == nil {
		ctx = audio.NewContext(DefaultSampleRate)
	}
	return &MusicManager{
		audioCtx:   ctx,
		wadFile:    w,
		volume:     0.7, // Default 70% volume
		sampleRate: DefaultSampleRate,
	}
}

// SetWAD updates the active WAD container used for loading music lumps.
func (m *MusicManager) SetWAD(w *wad.WAD) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wadFile = w
}

// Volume returns the current music playback volume (0.0 to 1.0).
func (m *MusicManager) Volume() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.volume
}

// SetVolume sets the music volume, clamped between 0.0 and 1.0.
func (m *MusicManager) SetVolume(v float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if v < 0.0 {
		v = 0.0
	} else if v > 1.0 {
		v = 1.0
	}
	m.volume = v
	if m.player != nil && !m.muted {
		m.player.SetVolume(v)
	}
}

// CurrentTrack returns the name of the currently active music lump.
func (m *MusicManager) CurrentTrack() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.currentTrack
}

// IsPlaying reports whether a music track is currently active and playing.
func (m *MusicManager) IsPlaying() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.player != nil && m.player.IsPlaying()
}

// Stop stops the currently playing music track and closes the player.
func (m *MusicManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.player != nil {
		m.player.Close()
		m.player = nil
	}
	m.currentTrack = ""
}

// Pause pauses the currently active music track.
func (m *MusicManager) Pause() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.player != nil && m.player.IsPlaying() {
		m.player.Pause()
	}
}

// Resume resumes playing the paused music track.
func (m *MusicManager) Resume() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.player != nil && !m.player.IsPlaying() {
		m.player.Play()
	}
}

// PlayMapMusic resolves the default music track for the given map name and starts playback.
func (m *MusicManager) PlayMapMusic(mapName string) error {
	track := wad.MapToMusic(mapName)
	return m.PlayMusic(track)
}

// PlayMusic loads and starts looping the specified music lump from the WAD container.
func (m *MusicManager) PlayMusic(lumpName string) error {
	upper := strings.ToUpper(strings.TrimSpace(lumpName))
	if upper == "" {
		m.Stop()
		return nil
	}

	m.mu.Lock()
	// If already playing this track, don't restart it
	if m.currentTrack == upper && m.player != nil && m.player.IsPlaying() {
		m.mu.Unlock()
		return nil
	}

	if m.wadFile == nil {
		m.mu.Unlock()
		return fmt.Errorf("cannot play music %s: no WAD file loaded", upper)
	}

	data, format, err := m.wadFile.GetMusicLump(upper)
	if err != nil {
		m.mu.Unlock()
		return fmt.Errorf("failed to retrieve music lump %s: %w", upper, err)
	}

	m.mu.Unlock()

	stream, streamLen, err := DecodeStream(data, m.sampleRate)
	if err != nil {
		return fmt.Errorf("failed to decode music lump %s (%s): %w", upper, format, err)
	}

	loopStream := audio.NewInfiniteLoop(stream, streamLen)

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.player != nil {
		m.player.Close()
		m.player = nil
	}

	player, err := m.audioCtx.NewPlayer(loopStream)
	if err != nil {
		return fmt.Errorf("failed to create audio player for %s: %w", upper, err)
	}

	player.SetVolume(m.volume)
	player.Play()

	m.player = player
	m.currentTrack = upper
	log.Printf("Playing music track: %s (%s)", upper, format)

	return nil
}
