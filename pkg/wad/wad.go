package wad

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	// ErrInvalidHeader is returned when the WAD header identification is not IWAD or PWAD.
	ErrInvalidHeader = errors.New("invalid WAD header identification")
	// ErrLumpNotFound is returned when a requested lump does not exist in the WAD.
	ErrLumpNotFound = errors.New("lump not found")
)

type rawHeader struct {
	Identification [4]byte
	NumLumps       int32
	InfoTableOfs   int32
}

type rawLumpEntry struct {
	FilePos int32
	Size    int32
	Name    [8]byte
}

// LumpInfo contains metadata about a lump in the WAD file.
type LumpInfo struct {
	Index   int
	Name    string
	FilePos int64
	Size    int64
}

// WAD represents a loaded Doom WAD (IWAD or PWAD) container.
type WAD struct {
	wadType string
	reader  io.ReaderAt
	closer  io.Closer
	lumps   []LumpInfo
	lumpMap map[string]int // maps uppercase lump name to last lump index (PWAD override semantics)
}

// Open opens a WAD file from the local file system.
func Open(filename string) (*WAD, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAD file %q: %w", filename, err)
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat WAD file %q: %w", filename, err)
	}

	w, err := OpenReader(file, stat.Size())
	if err != nil {
		file.Close()
		return nil, err
	}
	w.closer = file
	return w, nil
}

// OpenReader initializes a WAD container from an io.ReaderAt source.
func OpenReader(r io.ReaderAt, size int64) (*WAD, error) {
	var header rawHeader
	headerBuf := make([]byte, 12)
	if _, err := r.ReadAt(headerBuf, 0); err != nil {
		return nil, fmt.Errorf("failed to read WAD header: %w", err)
	}

	bufReader := bytes.NewReader(headerBuf)
	if err := binary.Read(bufReader, binary.LittleEndian, &header); err != nil {
		return nil, fmt.Errorf("failed to parse WAD header: %w", err)
	}

	id := string(header.Identification[:])
	if id != "IWAD" && id != "PWAD" {
		return nil, fmt.Errorf("%w: %q", ErrInvalidHeader, id)
	}

	if header.NumLumps < 0 || int64(header.InfoTableOfs)+int64(header.NumLumps)*16 > size {
		return nil, errors.New("invalid WAD directory offset or lump count")
	}

	tableData := make([]byte, header.NumLumps*16)
	if _, err := r.ReadAt(tableData, int64(header.InfoTableOfs)); err != nil {
		return nil, fmt.Errorf("failed to read WAD directory table: %w", err)
	}

	lumps := make([]LumpInfo, header.NumLumps)
	lumpMap := make(map[string]int, header.NumLumps)

	for i := 0; i < int(header.NumLumps); i++ {
		entryBytes := tableData[i*16 : (i+1)*16]
		var entry rawLumpEntry
		_ = binary.Read(bytes.NewReader(entryBytes), binary.LittleEndian, &entry)

		rawName := string(bytes.TrimRight(entry.Name[:], "\x00"))
		name := strings.ToUpper(rawName)

		lump := LumpInfo{
			Index:   i,
			Name:    name,
			FilePos: int64(entry.FilePos),
			Size:    int64(entry.Size),
		}
		lumps[i] = lump
		lumpMap[name] = i
	}

	return &WAD{
		wadType: id,
		reader:  r,
		lumps:   lumps,
		lumpMap: lumpMap,
	}, nil
}

// Close closes the underlying reader if it implements io.Closer.
func (w *WAD) Close() error {
	if w.closer != nil {
		return w.closer.Close()
	}
	return nil
}

// Type returns the WAD type ("IWAD" or "PWAD").
func (w *WAD) Type() string {
	return w.wadType
}

// NumLumps returns the number of lumps in the WAD.
func (w *WAD) NumLumps() int {
	return len(w.lumps)
}

// Lumps returns a copy of all lump information.
func (w *WAD) Lumps() []LumpInfo {
	res := make([]LumpInfo, len(w.lumps))
	copy(res, w.lumps)
	return res
}

// HasLump returns true if a lump with the given name exists.
func (w *WAD) HasLump(name string) bool {
	_, ok := w.lumpMap[strings.ToUpper(name)]
	return ok
}

// GetLumpIndex returns the index of the named lump, or -1 if not found.
func (w *WAD) GetLumpIndex(name string) int {
	if idx, ok := w.lumpMap[strings.ToUpper(name)]; ok {
		return idx
	}
	return -1
}

// GetLump reads and returns the full data of the named lump.
func (w *WAD) GetLump(name string) ([]byte, error) {
	idx := w.GetLumpIndex(name)
	if idx < 0 {
		return nil, fmt.Errorf("%w: %s", ErrLumpNotFound, name)
	}
	return w.GetLumpByIndex(idx)
}

// GetLumpByIndex reads and returns the data for the lump at the specified index.
func (w *WAD) GetLumpByIndex(index int) ([]byte, error) {
	if index < 0 || index >= len(w.lumps) {
		return nil, fmt.Errorf("lump index %d out of bounds (total %d)", index, len(w.lumps))
	}

	lump := w.lumps[index]
	if lump.Size == 0 {
		return []byte{}, nil
	}

	data := make([]byte, lump.Size)
	n, err := w.reader.ReadAt(data, lump.FilePos)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read lump %s data: %w", lump.Name, err)
	}
	if int64(n) < lump.Size {
		return nil, fmt.Errorf("short read for lump %s (expected %d, got %d)", lump.Name, lump.Size, n)
	}

	return data, nil
}
