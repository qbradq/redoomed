package wad

import (
	"bytes"
	"fmt"
	"strings"
)

// MusicFormat indicates the underlying audio/music format of a WAD lump.
type MusicFormat int

const (
	// MusicFormatUnknown indicates an unrecognized music format.
	MusicFormatUnknown MusicFormat = iota
	// MusicFormatMUS indicates classic Doom DMX MUS format ("MUS\x1a").
	MusicFormatMUS
	// MusicFormatMIDI indicates Standard MIDI File format ("MThd").
	MusicFormatMIDI
	// MusicFormatVorbis indicates Ogg Vorbis digital audio ("OggS").
	MusicFormatVorbis
	// MusicFormatMP3 indicates MP3 audio ("ID3" or sync frame).
	MusicFormatMP3
)

func (f MusicFormat) String() string {
	switch f {
	case MusicFormatMUS:
		return "MUS"
	case MusicFormatMIDI:
		return "MIDI"
	case MusicFormatVorbis:
		return "Ogg Vorbis"
	case MusicFormatMP3:
		return "MP3"
	default:
		return "Unknown"
	}
}

// DetectMusicFormat inspects the header bytes of lump data to determine its music/audio format.
func DetectMusicFormat(data []byte) MusicFormat {
	if len(data) < 4 {
		return MusicFormatUnknown
	}

	// 1. Doom MUS format: starts with "MUS\x1a" (0x1A4D5553)
	if bytes.HasPrefix(data, []byte("MUS\x1a")) {
		return MusicFormatMUS
	}

	// 2. Standard MIDI File: starts with "MThd"
	if bytes.HasPrefix(data, []byte("MThd")) {
		return MusicFormatMIDI
	}

	// 3. Ogg Vorbis: starts with "OggS"
	if bytes.HasPrefix(data, []byte("OggS")) {
		return MusicFormatVorbis
	}

	// 4. MP3 with ID3 tag: starts with "ID3"
	if bytes.HasPrefix(data, []byte("ID3")) {
		return MusicFormatMP3
	}

	// 5. MP3 raw sync frames (0xFF followed by 0xFB, 0xFA, or 0xF2/0xF3)
	if data[0] == 0xFF && (data[1]&0xE0) == 0xE0 {
		return MusicFormatMP3
	}

	return MusicFormatUnknown
}

// doom2MusicMap maps Doom 2 map numbers (1..32) to their standard music lump names.
var doom2MusicMap = [33]string{
	"",          // index 0 unused
	"D_RUNNIN",  // MAP01
	"D_STALKS",  // MAP02
	"D_COUNTD",  // MAP03
	"D_BETWEE",  // MAP04
	"D_DOOM",    // MAP05
	"D_THE_DA",  // MAP06
	"D_SHAWN",   // MAP07
	"D_DDTBLU",  // MAP08
	"D_IN_CIT",  // MAP09
	"D_DEAD",    // MAP10
	"D_STLKS2",  // MAP11
	"D_THEDA2",  // MAP12
	"D_DOOM2",   // MAP13
	"D_DDTBL2",  // MAP14
	"D_RUNNI2",  // MAP15
	"D_DEAD2",   // MAP16
	"D_STLKS3",  // MAP17
	"D_ROMERO",  // MAP18
	"D_SHAWN2",  // MAP19
	"D_MESSAG",  // MAP20
	"D_COUNT2",  // MAP21
	"D_DDTBL3",  // MAP22
	"D_AMPIE",   // MAP23
	"D_THEDA3",  // MAP24
	"D_ADRIAN",  // MAP25
	"D_MESSG2",  // MAP26
	"D_ROMER2",  // MAP27
	"D_TENSE",   // MAP28
	"D_SHAWN3",  // MAP29
	"D_OPENIN",  // MAP30
	"D_EVIL",    // MAP31 (Secret)
	"D_ULTIMA",  // MAP32 (Super Secret)
}

// MapToMusic resolves the standard Doom music lump name for a given map or screen identifier.
func MapToMusic(mapName string) string {
	upper := strings.ToUpper(strings.TrimSpace(mapName))
	if upper == "" {
		return "D_DM2TTL"
	}

	// Doom 2 Map: MAP01 - MAP32
	if strings.HasPrefix(upper, "MAP") && len(upper) >= 5 {
		var mapNum int
		if _, err := fmt.Sscanf(upper[3:], "%d", &mapNum); err == nil {
			if mapNum >= 1 && mapNum < len(doom2MusicMap) {
				return doom2MusicMap[mapNum]
			}
		}
		return "D_RUNNIN"
	}

	// Doom 1 Map: E1M1 - E4M9
	if len(upper) == 4 && upper[0] == 'E' && upper[2] == 'M' {
		episode := upper[1] - '0'
		mission := upper[3] - '0'
		if episode >= 1 && episode <= 4 && mission >= 1 && mission <= 9 {
			return fmt.Sprintf("D_E%dM%d", episode, mission)
		}
	}

	// UI & Special screens
	switch upper {
	case "TITLE", "TITLEPIC", "INTRO":
		return "D_DM2TTL"
	case "INTER", "INTERPIC", "VICTORY":
		return "D_DM2INT"
	case "READ_M", "TEXT":
		return "D_READ_M"
	}

	// If already in D_ format, return as is
	if strings.HasPrefix(upper, "D_") {
		return upper
	}

	return "D_" + upper
}

// GetMusicLump retrieves the lump data and detects the music format for the given music lump name.
func (w *WAD) GetMusicLump(name string) ([]byte, MusicFormat, error) {
	upper := strings.ToUpper(strings.TrimSpace(name))
	data, err := w.GetLump(upper)
	if err != nil {
		return nil, MusicFormatUnknown, err
	}

	format := DetectMusicFormat(data)
	return data, format, nil
}
