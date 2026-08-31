package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	// ErrInvalidMUS indicates an invalid or corrupted MUS data stream.
	ErrInvalidMUS = errors.New("invalid MUS format")
)

type musHeader struct {
	Magic       [4]byte // "MUS\x1a"
	ScoreLen    uint16
	ScoreStart  uint16
	Channels    uint16
	SecChannels uint16
	InstrCount  uint16
	Dummy       uint16
}

// ConvertMUSToMIDI converts a Doom MUS lump byte slice into a Standard MIDI File (SMF Type 0) byte slice.
func ConvertMUSToMIDI(musData []byte) ([]byte, error) {
	if len(musData) < 16 {
		return nil, fmt.Errorf("%w: header too short (%d bytes)", ErrInvalidMUS, len(musData))
	}

	var header musHeader
	reader := bytes.NewReader(musData[:16])
	if err := binary.Read(reader, binary.LittleEndian, &header); err != nil {
		return nil, fmt.Errorf("%w: failed to read header: %w", ErrInvalidMUS, err)
	}

	if string(header.Magic[:]) != "MUS\x1a" {
		return nil, fmt.Errorf("%w: magic %q does not match MUS\\x1a", ErrInvalidMUS, header.Magic)
	}

	start := int(header.ScoreStart)
	if start < 16 || start >= len(musData) {
		return nil, fmt.Errorf("%w: invalid score start offset %d", ErrInvalidMUS, start)
	}

	// Map MUS channel (0-15) to MIDI channel (0-15)
	// MUS ch 15 is percussion (MIDI ch 9)
	channelMap := [16]byte{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 10, 11, 12, 13, 14, 15, 9,
	}

	// MUS controller number to MIDI controller number
	ctrlMap := [10]byte{
		0,  // 0: Program Change (handled specially)
		0,  // 1: Bank Select (CC 0)
		1,  // 2: Modulation (CC 1)
		7,  // 3: Volume (CC 7)
		10, // 4: Pan (CC 10)
		11, // 5: Expression (CC 11)
		91, // 6: Reverb (CC 91)
		93, // 7: Chorus (CC 93)
		64, // 8: Sustain Pedal (CC 64)
		67, // 9: Soft Pedal (CC 67)
	}

	// Track channel volume states (default 100)
	chanVolume := [16]byte{
		100, 100, 100, 100, 100, 100, 100, 100,
		100, 100, 100, 100, 100, 100, 100, 100,
	}

	trackBuf := new(bytes.Buffer)
	pos := start
	var currentDelta uint32
	scoreEnd := false

	for pos < len(musData) && !scoreEnd {
		eventByte := musData[pos]
		pos++

		lastBit := (eventByte & 0x80) != 0
		eventType := (eventByte >> 4) & 0x07
		musChan := eventByte & 0x0F
		midChan := channelMap[musChan]

		switch eventType {
		case 0:
			// Release Note
			if pos >= len(musData) {
				break
			}
			note := musData[pos] & 0x7F
			pos++

			writeVarLen(trackBuf, currentDelta)
			currentDelta = 0
			trackBuf.WriteByte(0x80 | midChan)
			trackBuf.WriteByte(note)
			trackBuf.WriteByte(64) // Default release velocity

		case 1:
			// Play Note
			if pos >= len(musData) {
				break
			}
			noteByte := musData[pos]
			pos++

			note := noteByte & 0x7F
			if noteByte&0x80 != 0 {
				if pos < len(musData) {
					chanVolume[musChan] = musData[pos] & 0x7F
					pos++
				}
			}

			writeVarLen(trackBuf, currentDelta)
			currentDelta = 0
			trackBuf.WriteByte(0x90 | midChan)
			trackBuf.WriteByte(note)
			trackBuf.WriteByte(chanVolume[musChan])

		case 2:
			// Pitch Wheel
			if pos >= len(musData) {
				break
			}
			wheelVal := musData[pos]
			pos++

			// MUS pitch wheel 0..255 -> MIDI 14-bit pitch bend (0..16383, center 8192)
			bend := uint16(wheelVal) * 64
			lsb := byte(bend & 0x7F)
			msb := byte((bend >> 7) & 0x7F)

			writeVarLen(trackBuf, currentDelta)
			currentDelta = 0
			trackBuf.WriteByte(0xE0 | midChan)
			trackBuf.WriteByte(lsb)
			trackBuf.WriteByte(msb)

		case 3:
			// System Event
			if pos >= len(musData) {
				break
			}
			ctrl := musData[pos] & 0x7F
			pos++

			if ctrl == 10 {
				// All Sounds Off (CC 120)
				writeVarLen(trackBuf, currentDelta)
				currentDelta = 0
				trackBuf.WriteByte(0xB0 | midChan)
				trackBuf.WriteByte(120)
				trackBuf.WriteByte(0)
			} else if ctrl == 11 {
				// All Notes Off (CC 123)
				writeVarLen(trackBuf, currentDelta)
				currentDelta = 0
				trackBuf.WriteByte(0xB0 | midChan)
				trackBuf.WriteByte(123)
				trackBuf.WriteByte(0)
			} else if ctrl == 14 {
				// Reset All Controllers (CC 121)
				writeVarLen(trackBuf, currentDelta)
				currentDelta = 0
				trackBuf.WriteByte(0xB0 | midChan)
				trackBuf.WriteByte(121)
				trackBuf.WriteByte(0)
			}

		case 4:
			// Change Controller
			if pos+1 >= len(musData) {
				pos = len(musData)
				break
			}
			ctrl := musData[pos] & 0x7F
			val := musData[pos+1] & 0x7F
			pos += 2

			if ctrl == 0 {
				// Program Change (Instrument)
				writeVarLen(trackBuf, currentDelta)
				currentDelta = 0
				trackBuf.WriteByte(0xC0 | midChan)
				trackBuf.WriteByte(val)
			} else if ctrl < 10 {
				midCtrl := ctrlMap[ctrl]
				writeVarLen(trackBuf, currentDelta)
				currentDelta = 0
				trackBuf.WriteByte(0xB0 | midChan)
				trackBuf.WriteByte(midCtrl)
				trackBuf.WriteByte(val)
			}

		case 5:
			// Score End
			scoreEnd = true

		default:
			// Unused event types
		}

		if lastBit {
			// Read variable-length delta time
			var delta uint32
			for pos < len(musData) {
				b := musData[pos]
				pos++
				delta = (delta << 7) | uint32(b&0x7F)
				if (b & 0x80) == 0 {
					break
				}
			}
			currentDelta += delta
		}
	}

	// Write End of Track meta event (FF 2F 00)
	writeVarLen(trackBuf, currentDelta)
	trackBuf.Write([]byte{0xFF, 0x2F, 0x00})

	// Build Standard MIDI File (SMF Type 0, 1 track, 70 ticks per quarter note -> 140 ticks/sec at default 120 BPM)
	out := new(bytes.Buffer)

	// MThd chunk
	out.WriteString("MThd")
	_ = binary.Write(out, binary.BigEndian, uint32(6))  // Header length
	_ = binary.Write(out, binary.BigEndian, uint16(0))  // Format 0 (single track)
	_ = binary.Write(out, binary.BigEndian, uint16(1))  // 1 Track
	_ = binary.Write(out, binary.BigEndian, uint16(70)) // Division: 70 ticks/quarter note

	// MTrk chunk
	out.WriteString("MTrk")
	trackBytes := trackBuf.Bytes()
	_ = binary.Write(out, binary.BigEndian, uint32(len(trackBytes)))
	out.Write(trackBytes)

	return out.Bytes(), nil
}

// writeVarLen writes a MIDI variable-length quantity.
func writeVarLen(buf *bytes.Buffer, value uint32) {
	buffer := value & 0x7F
	for {
		value >>= 7
		if value == 0 {
			break
		}
		buffer <<= 8
		buffer |= ((value & 0x7F) | 0x80)
	}

	for {
		buf.WriteByte(byte(buffer & 0xFF))
		if (buffer & 0x80) != 0 {
			buffer >>= 8
		} else {
			break
		}
	}
}
