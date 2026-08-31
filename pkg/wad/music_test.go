package wad

import (
	"os"
	"testing"
)

func TestDetectMusicFormat(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected MusicFormat
	}{
		{
			name:     "MUS format",
			data:     []byte{'M', 'U', 'S', 0x1A, 0x00, 0x10},
			expected: MusicFormatMUS,
		},
		{
			name:     "MIDI format",
			data:     []byte{'M', 'T', 'h', 'd', 0x00, 0x00, 0x00, 0x06},
			expected: MusicFormatMIDI,
		},
		{
			name:     "Ogg Vorbis format",
			data:     []byte{'O', 'g', 'g', 'S', 0x00, 0x02},
			expected: MusicFormatVorbis,
		},
		{
			name:     "MP3 ID3 format",
			data:     []byte{'I', 'D', '3', 0x03, 0x00, 0x00},
			expected: MusicFormatMP3,
		},
		{
			name:     "MP3 sync frame format",
			data:     []byte{0xFF, 0xFB, 0x90, 0x64},
			expected: MusicFormatMP3,
		},
		{
			name:     "Unknown / short format",
			data:     []byte{0x01, 0x02},
			expected: MusicFormatUnknown,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DetectMusicFormat(tc.data)
			if got != tc.expected {
				t.Errorf("DetectMusicFormat() = %v (%s), expected %v (%s)", got, got, tc.expected, tc.expected)
			}
		})
	}
}

func TestMapToMusic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"MAP01", "D_RUNNIN"},
		{"MAP02", "D_STALKS"},
		{"MAP07", "D_SHAWN"},
		{"MAP10", "D_DEAD"},
		{"MAP30", "D_OPENIN"},
		{"MAP31", "D_EVIL"},
		{"MAP32", "D_ULTIMA"},
		{"E1M1", "D_E1M1"},
		{"E2M3", "D_E2M3"},
		{"E4M9", "D_E4M9"},
		{"TITLE", "D_DM2TTL"},
		{"INTERPIC", "D_DM2INT"},
		{"D_RUNNIN", "D_RUNNIN"},
		{"", "D_DM2TTL"},
	}

	for _, tc := range tests {
		got := MapToMusic(tc.input)
		if got != tc.expected {
			t.Errorf("MapToMusic(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestGetMusicLumpFreedoom(t *testing.T) {
	wadPath := "../../freedoom2.wad"
	if _, err := os.Stat(wadPath); os.IsNotExist(err) {
		wadPath = "freedoom2.wad"
		if _, err := os.Stat(wadPath); os.IsNotExist(err) {
			t.Skip("freedoom2.wad not found, skipping")
		}
	}

	w, err := Open(wadPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer w.Close()

	// Freedoom 2 has D_RUNNIN
	data, format, err := w.GetMusicLump("D_RUNNIN")
	if err != nil {
		t.Fatalf("GetMusicLump(D_RUNNIN) failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty data for D_RUNNIN")
	}

	if format != MusicFormatMIDI && format != MusicFormatMUS && format != MusicFormatVorbis {
		t.Errorf("unexpected music format for D_RUNNIN: %v", format)
	}
}
