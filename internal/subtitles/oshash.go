package subtitles

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// OpenSubtitles identifies a video by a cheap hash: the file size plus the little-endian
// 64-bit word sum of its first and last 64 KiB. A subtitle uploaded against the exact same
// release carries the same hash, and "moviehash_match" results are the ones that are
// actually in sync — a title/season/episode match is the right SHOW, but a different cut
// or frame rate drifts by seconds. Preferring hash matches is the single biggest win in
// subtitle quality a provider search can make.
const osHashChunk = 64 * 1024

// osHash computes the OpenSubtitles hash of the file at path. Returns "" (no error) for a
// file too small to hash the way the protocol expects; the caller just searches without it.
func osHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return "", err
	}
	size := st.Size()
	if size < osHashChunk*2 {
		return "", nil
	}
	var sum uint64 = uint64(size)
	add := func(off int64) error {
		buf := make([]byte, osHashChunk)
		if _, err := f.ReadAt(buf, off); err != nil && err != io.EOF {
			return err
		}
		for i := 0; i+8 <= len(buf); i += 8 {
			sum += binary.LittleEndian.Uint64(buf[i : i+8])
		}
		return nil
	}
	if err := add(0); err != nil {
		return "", err
	}
	if err := add(size - osHashChunk); err != nil {
		return "", err
	}
	return fmt.Sprintf("%016x", sum), nil
}
