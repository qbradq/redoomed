package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

const (
	// DefaultSampleRate is the standard audio playback sample rate.
	DefaultSampleRate = 44100
	// BytesPerSample is 4 bytes (16-bit stereo = 2 channels * 2 bytes).
	BytesPerSample = 4
	// MaxVoices is the maximum simultaneous polyphony for the MIDI synthesizer.
	MaxVoices = 64
)

// DecodeStream decodes audio lump data (Vorbis, MP3, MIDI, or MUS) into a 16-bit stereo PCM stream.
func DecodeStream(data []byte, sampleRate int) (io.ReadSeeker, int64, error) {
	if len(data) < 4 {
		return nil, 0, errors.New("audio data too short")
	}

	// 1. Ogg Vorbis
	if bytes.HasPrefix(data, []byte("OggS")) {
		stream, err := vorbis.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
		if err != nil {
			return nil, 0, fmt.Errorf("failed to decode Ogg Vorbis stream: %w", err)
		}
		return stream, stream.Length(), nil
	}

	// 2. MP3 (ID3 or sync word)
	if bytes.HasPrefix(data, []byte("ID3")) || (data[0] == 0xFF && (data[1]&0xE0) == 0xE0) {
		stream, err := mp3.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
		if err != nil {
			return nil, 0, fmt.Errorf("failed to decode MP3 stream: %w", err)
		}
		return stream, stream.Length(), nil
	}

	// 3. MUS -> Convert to MIDI first
	midiData := data
	if bytes.HasPrefix(data, []byte("MUS\x1a")) {
		converted, err := ConvertMUSToMIDI(data)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert MUS to MIDI: %w", err)
		}
		midiData = converted
	}

	// 4. Standard MIDI
	if bytes.HasPrefix(midiData, []byte("MThd")) {
		synth, err := NewMIDISynth(midiData, sampleRate)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to initialize MIDI synth: %w", err)
		}
		pcmBytes, err := synth.RenderAll()
		if err != nil {
			return nil, 0, fmt.Errorf("failed to render MIDI to PCM: %w", err)
		}
		stream := bytes.NewReader(pcmBytes)
		return stream, int64(len(pcmBytes)), nil
	}

	return nil, 0, errors.New("unrecognized audio stream format")
}

// midiEvent represents a timed MIDI message.
type midiEvent struct {
	tick     int64
	status   byte
	data1    byte
	data2    byte
	isMeta   bool
	metaType byte
	metaData []byte
}

// voice represents a single active audio synthesizer voice.
type voice struct {
	active     bool
	channel    byte
	note       byte
	velocity   float64
	frequency  float64
	phase      float64
	phaseMod   float64
	envPhase   int // 0: attack, 1: decay, 2: sustain, 3: release, 4: off
	envLevel   float64
	program    byte
	isDrum     bool
	drumType   byte
	pan        float64 // 0.0 (left) to 1.0 (right)
	sampleStep float64
	ageSamples int64
}

// MIDISynth parses standard MIDI files and synthesizes multi-channel 16-bit PCM audio.
type MIDISynth struct {
	sampleRate  int
	division    int
	events      []midiEvent
	totalTicks  int64
	totalTimeMs float64
}

// NewMIDISynth creates and parses a MIDI stream into a synthesizer instance.
func NewMIDISynth(midiData []byte, sampleRate int) (*MIDISynth, error) {
	if len(midiData) < 14 || string(midiData[:4]) != "MThd" {
		return nil, errors.New("invalid MIDI header")
	}

	format := binary.BigEndian.Uint16(midiData[8:10])
	numTracks := int(binary.BigEndian.Uint16(midiData[10:12]))
	division := int(binary.BigEndian.Uint16(midiData[12:14]))

	if division <= 0 {
		division = 140
	}

	synth := &MIDISynth{
		sampleRate: sampleRate,
		division:   division,
	}

	var allEvents []midiEvent
	offset := 14

	for t := 0; t < numTracks && offset < len(midiData); t++ {
		for offset+8 <= len(midiData) && string(midiData[offset:offset+4]) != "MTrk" {
			chunkLen := int(binary.BigEndian.Uint32(midiData[offset+4 : offset+8]))
			offset += 8 + chunkLen
		}

		if offset+8 > len(midiData) {
			break
		}

		trackLen := int(binary.BigEndian.Uint32(midiData[offset+4 : offset+8]))
		offset += 8
		trackEnd := offset + trackLen
		if trackEnd > len(midiData) {
			trackEnd = len(midiData)
		}

		trackEvents, err := parseTrackEvents(midiData[offset:trackEnd])
		if err == nil {
			allEvents = append(allEvents, trackEvents...)
		}

		offset = trackEnd
	}

	if len(allEvents) == 0 {
		return nil, errors.New("no valid MIDI events found")
	}

	// Sort events by tick (for Format 1 multi-track files)
	if format == 1 {
		sortMIDIEvents(allEvents)
	}

	synth.events = allEvents
	if len(allEvents) > 0 {
		synth.totalTicks = allEvents[len(allEvents)-1].tick
	}

	return synth, nil
}

func parseTrackEvents(trackData []byte) ([]midiEvent, error) {
	var events []midiEvent
	pos := 0
	var currentTick int64
	var runningStatus byte

	for pos < len(trackData) {
		delta, n := readVarLenMIDI(trackData[pos:])
		pos += n
		currentTick += int64(delta)

		if pos >= len(trackData) {
			break
		}

		status := trackData[pos]
		if status < 0x80 {
			// Running status
			status = runningStatus
		} else {
			pos++
			if status < 0xF0 {
				runningStatus = status
			}
		}

		if status == 0xFF {
			// Meta event
			if pos >= len(trackData) {
				break
			}
			metaType := trackData[pos]
			pos++
			metaLen, n := readVarLenMIDI(trackData[pos:])
			pos += n
			var metaData []byte
			if pos+int(metaLen) <= len(trackData) {
				metaData = trackData[pos : pos+int(metaLen)]
				pos += int(metaLen)
			}
			events = append(events, midiEvent{
				tick:     currentTick,
				status:   status,
				isMeta:   true,
				metaType: metaType,
				metaData: metaData,
			})
			if metaType == 0x2F {
				// End of track
				break
			}
		} else if status == 0xF0 || status == 0xF7 {
			// Sysex event
			sysexLen, n := readVarLenMIDI(trackData[pos:])
			pos += n + int(sysexLen)
		} else {
			// Standard channel event
			eventType := status & 0xF0
			var d1, d2 byte
			if pos < len(trackData) {
				d1 = trackData[pos]
				pos++
			}
			if eventType != 0xC0 && eventType != 0xD0 {
				if pos < len(trackData) {
					d2 = trackData[pos]
					pos++
				}
			}
			events = append(events, midiEvent{
				tick:   currentTick,
				status: status,
				data1:  d1,
				data2:  d2,
			})
		}
	}

	return events, nil
}

func readVarLenMIDI(data []byte) (uint32, int) {
	var val uint32
	for i, b := range data {
		val = (val << 7) | uint32(b&0x7F)
		if (b & 0x80) == 0 {
			return val, i + 1
		}
	}
	return val, len(data)
}

func sortMIDIEvents(events []midiEvent) {
	// Simple stable insertion sort
	for i := 1; i < len(events); i++ {
		key := events[i]
		j := i - 1
		for j >= 0 && events[j].tick > key.tick {
			events[j+1] = events[j]
			j--
		}
		events[j+1] = key
	}
}

// RenderAll renders the entire MIDI sequence into a 16-bit 44.1kHz stereo PCM byte buffer.
func (s *MIDISynth) RenderAll() ([]byte, error) {
	voices := make([]voice, MaxVoices)
	chanPrograms := make([]byte, 16)
	chanVolumes := make([]float64, 16)
	chanPans := make([]float64, 16)
	chanPitchBends := make([]float64, 16) // semitone offset

	for i := range chanVolumes {
		chanVolumes[i] = 0.8
		chanPans[i] = 0.5
	}

	tempoMicros := 500000.0 // Default 120 BPM (500,000 µs/beat)
	secondsPerTick := (tempoMicros / 1000000.0) / float64(s.division)

	eventIdx := 0
	var currentTick int64
	var currentSample int64
	out := new(bytes.Buffer)

	// Note to frequency helper
	noteFreq := func(note byte, bend float64) float64 {
		return 440.0 * math.Pow(2.0, (float64(note)+bend-69.0)/12.0)
	}

	for eventIdx < len(s.events) || hasActiveVoices(voices) {
		// Process all events scheduled at or before currentTick
		for eventIdx < len(s.events) && s.events[eventIdx].tick <= currentTick {
			ev := s.events[eventIdx]
			eventIdx++

			if ev.isMeta {
				if ev.metaType == 0x51 && len(ev.metaData) >= 3 {
					// Set Tempo
					tempo := float64(int(ev.metaData[0])<<16 | int(ev.metaData[1])<<8 | int(ev.metaData[2]))
					if tempo > 0 {
						tempoMicros = tempo
						secondsPerTick = (tempoMicros / 1000000.0) / float64(s.division)
					}
				}
				continue
			}

			ch := ev.status & 0x0F
			cmd := ev.status & 0xF0

			switch cmd {
			case 0x80:
				// Note Off
				for i := range voices {
					if voices[i].active && voices[i].channel == ch && voices[i].note == ev.data1 {
						voices[i].envPhase = 3 // Release
					}
				}
			case 0x90:
				// Note On (velocity > 0) or Note Off (velocity == 0)
				if ev.data2 == 0 {
					for i := range voices {
						if voices[i].active && voices[i].channel == ch && voices[i].note == ev.data1 {
							voices[i].envPhase = 3
						}
					}
				} else {
					// Allocate voice
					idx := allocVoice(voices)
					isPerc := (ch == 9)
					freq := noteFreq(ev.data1, chanPitchBends[ch])
					if isPerc {
						freq = drumBaseFreq(ev.data1)
					}

					voices[idx] = voice{
						active:     true,
						channel:    ch,
						note:       ev.data1,
						velocity:   float64(ev.data2) / 127.0,
						frequency:  freq,
						phase:      0,
						phaseMod:   0,
						envPhase:   0, // Attack
						envLevel:   0.01,
						program:    chanPrograms[ch],
						isDrum:     isPerc,
						drumType:   ev.data1,
						pan:        chanPans[ch],
						sampleStep: freq * 2.0 * math.Pi / float64(s.sampleRate),
					}
				}
			case 0xC0:
				// Program Change
				chanPrograms[ch] = ev.data1
			case 0xE0:
				// Pitch Bend
				raw := int(ev.data1) | (int(ev.data2) << 7)
				bendSemitones := (float64(raw-8192) / 8192.0) * 2.0 // +/- 2 semitones
				chanPitchBends[ch] = bendSemitones
				for i := range voices {
					if voices[i].active && voices[i].channel == ch && !voices[i].isDrum {
						voices[i].frequency = noteFreq(voices[i].note, bendSemitones)
						voices[i].sampleStep = voices[i].frequency * 2.0 * math.Pi / float64(s.sampleRate)
					}
				}
			case 0xB0:
				// Control Change
				switch ev.data1 {
				case 7:
					chanVolumes[ch] = float64(ev.data2) / 127.0
				case 10:
					chanPans[ch] = float64(ev.data2) / 127.0
				case 120, 123:
					for i := range voices {
						if voices[i].channel == ch {
							voices[i].active = false
						}
					}
				}
			}
		}

		// Calculate how many samples until next event
		var nextTick int64
		if eventIdx < len(s.events) {
			nextTick = s.events[eventIdx].tick
		} else {
			// Finish ring-out of active voices (max 2 seconds)
			nextTick = currentTick + int64(s.division*2)
		}

		samplesToRender := int(float64(nextTick-currentTick) * secondsPerTick * float64(s.sampleRate))
		if samplesToRender <= 0 {
			samplesToRender = 1
		}
		if samplesToRender > s.sampleRate*2 {
			samplesToRender = s.sampleRate * 2
		}

		// Render audio frame chunk
		chunk := renderVoiceSamples(voices, chanVolumes, samplesToRender)
		out.Write(chunk)

		currentSample += int64(samplesToRender)
		currentTick = nextTick
	}

	return out.Bytes(), nil
}

func hasActiveVoices(voices []voice) bool {
	for i := range voices {
		if voices[i].active {
			return true
		}
	}
	return false
}

func allocVoice(voices []voice) int {
	// Find inactive voice
	for i := range voices {
		if !voices[i].active {
			return i
		}
	}
	// Voice stealing: replace oldest voice
	oldest := 0
	var maxAge int64
	for i := range voices {
		if voices[i].ageSamples > maxAge {
			maxAge = voices[i].ageSamples
			oldest = i
		}
	}
	return oldest
}

func drumBaseFreq(note byte) float64 {
	switch note {
	case 35, 36: // Bass Drum
		return 60.0
	case 38, 40: // Snare Drum
		return 180.0
	case 42, 44, 46: // Hi-Hat
		return 800.0
	case 41, 43, 45, 47, 48, 50: // Toms
		return 120.0
	case 49, 57: // Crash Cymbal
		return 600.0
	case 51, 59: // Ride Cymbal
		return 500.0
	default:
		return 150.0
	}
}

// renderVoiceSamples generates 16-bit interleaved stereo PCM samples from active voices.
func renderVoiceSamples(voices []voice, chanVolumes []float64, numSamples int) []byte {
	buf := make([]byte, numSamples*BytesPerSample)

	for s := 0; s < numSamples; s++ {
		var leftSample, rightSample float64

		for i := range voices {
			v := &voices[i]
			if !v.active {
				continue
			}

			v.ageSamples++

			// Envelope processing
			switch v.envPhase {
			case 0: // Attack
				v.envLevel += 0.08
				if v.envLevel >= 1.0 {
					v.envLevel = 1.0
					v.envPhase = 1
				}
			case 1: // Decay
				decayRate := 0.00015
				if v.isDrum {
					decayRate = 0.003
				}
				v.envLevel -= decayRate
				if v.envLevel <= 0.65 && !v.isDrum {
					v.envLevel = 0.65
					v.envPhase = 2 // Sustain
				} else if v.envLevel <= 0.001 && v.isDrum {
					v.active = false
					continue
				}
			case 2: // Sustain
				// Slow decay in sustain
				v.envLevel *= 0.99998
				if v.envLevel < 0.05 {
					v.active = false
					continue
				}
			case 3: // Release
				v.envLevel -= 0.002
				if v.envLevel <= 0.001 {
					v.active = false
					continue
				}
			}

			// Generate waveform based on instrument family
			var wave float64
			if v.isDrum {
				// Percussion synthesis: noise + damped sine
				noise := (math.Mod(float64(v.ageSamples*1103515245+12345), 65536) / 32768.0) - 1.0
				sine := math.Sin(v.phase)
				wave = sine*0.6 + noise*0.4
			} else {
				// Melodic synthesis (rich FM synth with harmonics)
				prog := v.program
				switch prog / 8 {
				case 0, 1: // Piano / Chromatic: Warm Sine + 2nd harmonic
					wave = math.Sin(v.phase) + 0.3*math.Sin(v.phase*2.0)
				case 2, 3: // Organ / Guitar: FM Modulated Saw/Square
					wave = 0.6*math.Sin(v.phase) + 0.3*math.Sin(v.phase*3.0) + 0.1*math.Sin(v.phase*5.0)
				case 4: // Strings / Ensemble: Rich Sawtooth
					wave = (math.Mod(v.phase, 2*math.Pi)/math.Pi - 1.0) * 0.7
				case 5: // Brass: FM Brass Wave
					mod := math.Sin(v.phase * 2.0)
					wave = math.Sin(v.phase + mod*0.5)
				case 6: // Reed / Pipe: Clarinet / Flute
					wave = 0.8*math.Sin(v.phase) + 0.2*math.Sin(v.phase*3.0)
				case 7: // Synth Lead: Sharp Saw / Pulse
					wave = (math.Mod(v.phase, 2*math.Pi)/math.Pi - 1.0) * 0.8
				default:
					wave = math.Sin(v.phase)
				}
			}

			v.phase += v.sampleStep
			if v.phase > 2*math.Pi*1000.0 {
				v.phase = math.Mod(v.phase, 2*math.Pi)
			}

			vol := v.velocity * v.envLevel * chanVolumes[v.channel] * 0.35
			sampleVal := wave * vol

			// Stereo Panning
			leftPan := math.Cos(v.pan * math.Pi / 2.0)
			rightPan := math.Sin(v.pan * math.Pi / 2.0)

			leftSample += sampleVal * leftPan
			rightSample += sampleVal * rightPan
		}

		// Master soft clipping / saturation
		leftSample = math.Tanh(leftSample)
		rightSample = math.Tanh(rightSample)

		// Convert to 16-bit PCM (-32768 to 32767)
		leftInt := int16(leftSample * 32767.0)
		rightInt := int16(rightSample * 32767.0)

		idx := s * BytesPerSample
		buf[idx] = byte(leftInt & 0xFF)
		buf[idx+1] = byte((leftInt >> 8) & 0xFF)
		buf[idx+2] = byte(rightInt & 0xFF)
		buf[idx+3] = byte((rightInt >> 8) & 0xFF)
	}

	return buf
}
