package audio

import (
	"os"
	"testing"

	"github.com/qbradq/redoomed/pkg/wad"
)

func TestMUSToMIDIConversion(t *testing.T) {
	// Construct a minimal valid MUS header and score
	// Header: "MUS\x1a", scoreLen=6, scoreStart=16, channels=1, secChannels=0, instrCount=0, dummy=0
	musData := []byte{
		'M', 'U', 'S', 0x1A, // Magic
		0x07, 0x00, // ScoreLen (7 bytes)
		0x10, 0x00, // ScoreStart (16)
		0x01, 0x00, // Channels (1)
		0x00, 0x00, // SecChannels
		0x00, 0x00, // InstrCount
		0x00, 0x00, // Dummy
		// Score data at offset 16:
		// Play note event on ch 0: (type 1 << 4 | ch 0) = 0x10. Note=60 (middle C). Last bit set=0x90.
		0x90, 0x3C,
		// Delta time: 140 ticks = (1 << 7) | 12 -> 0x81, 0x0C
		0x81, 0x0C,
		// Release note event on ch 0: (type 0 << 4 | ch 0) = 0x00. Note=60. Last bit set=0x80.
		0x80, 0x3C,
		0x00, // Delta time 0
		// Score end event: (type 5 << 4) = 0x50
		0x50,
	}

	midiBytes, err := ConvertMUSToMIDI(musData)
	if err != nil {
		t.Fatalf("ConvertMUSToMIDI failed: %v", err)
	}

	if len(midiBytes) < 22 {
		t.Fatalf("MIDI data too short: %d bytes", len(midiBytes))
	}

	if string(midiBytes[:4]) != "MThd" {
		t.Errorf("expected MThd header, got %q", midiBytes[:4])
	}

	// Verify division in MIDI header is 70 (0x0046)
	division := int(midiBytes[12])<<8 | int(midiBytes[13])
	if division != 70 {
		t.Errorf("expected MIDI division 70, got %d", division)
	}

	// Render the 140-tick MUS stream
	stream, length, err := DecodeStream(musData, DefaultSampleRate)
	if err != nil {
		t.Fatalf("DecodeStream on MUS data failed: %v", err)
	}
	if stream == nil || length <= 0 {
		t.Fatalf("expected non-nil stream with positive length, got len=%d", length)
	}
	// 140 ticks at 140Hz = 1.0 second (44100 samples) + 1.0 second ringout (44100 samples) = 88200 samples = 352800 bytes
	expectedSamples := 44100 + 44100
	expectedBytes := int64(expectedSamples * BytesPerSample)
	if length != expectedBytes {
		t.Errorf("expected %d bytes (2.0s PCM), got %d bytes (%f s)", expectedBytes, length, float64(length)/(float64(DefaultSampleRate)*BytesPerSample))
	}
}

func TestMIDISynthRender(t *testing.T) {
	wadPath := "../../freedoom2.wad"
	if _, err := os.Stat(wadPath); os.IsNotExist(err) {
		wadPath = "freedoom2.wad"
		if _, err := os.Stat(wadPath); os.IsNotExist(err) {
			t.Skip("freedoom2.wad not found, skipping")
		}
	}

	w, err := wad.Open(wadPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer w.Close()

	data, _, err := w.GetMusicLump("D_RUNNIN")
	if err != nil {
		t.Fatalf("GetMusicLump(D_RUNNIN) failed: %v", err)
	}

	synth, err := NewMIDISynth(data, DefaultSampleRate)
	if err != nil {
		t.Fatalf("NewMIDISynth failed: %v", err)
	}

	pcm, err := synth.RenderAll()
	if err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	if len(pcm) == 0 {
		t.Error("expected non-empty rendered PCM byte buffer")
	}
}

func TestMusicManagerVolumeAndState(t *testing.T) {
	mgr := NewMusicManager(nil)
	if mgr == nil {
		t.Fatal("expected non-nil MusicManager")
	}

	if mgr.Volume() != 0.7 {
		t.Errorf("expected default volume 0.7, got %f", mgr.Volume())
	}

	mgr.SetVolume(0.4)
	if mgr.Volume() != 0.4 {
		t.Errorf("expected volume 0.4, got %f", mgr.Volume())
	}

	// Clamp tests
	mgr.SetVolume(1.5)
	if mgr.Volume() != 1.0 {
		t.Errorf("expected clamped volume 1.0, got %f", mgr.Volume())
	}

	mgr.SetVolume(-0.5)
	if mgr.Volume() != 0.0 {
		t.Errorf("expected clamped volume 0.0, got %f", mgr.Volume())
	}
}
