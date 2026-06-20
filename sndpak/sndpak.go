// Package sndpak bundles several sound files into a single ".pak" blob with a
// light, deliberately reversible obfuscation pass.
//
// The goal is NOT cryptographic security. It only prevents the trivial "rename
// the committed file to .wav and play it" path, so that licensed sound assets
// pulled from free-asset sites are not directly reusable when this repository is
// published. The keystream seed lives in source on purpose; anyone reading this
// package can recover the originals. Treat this as a speed bump that respects
// asset licenses (no raw, ready-to-use files in the repo), not as protection.
//
// The packer tool (tools/sndpak) writes the .pak; the game embeds and unpacks it
// at load time. Both import this package, which has no Ebiten dependency so the
// tool builds headless.
package sndpak

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// magic identifies a pak file. The header stays in cleartext to identify the
// format; everything after it (the entry table and blobs, including any RIFF
// headers) is obfuscated so a casual `strings`/`binwalk` finds nothing usable.
var magic = []byte("SPK1")

// obfSeed seeds the keystream. Non-zero (required by xorshift) and otherwise
// arbitrary. Changing it invalidates previously packed files.
const obfSeed uint32 = 0x9E3779B9

// Entry is one named blob in a pak, e.g. {"fire", <wav bytes>}.
type Entry struct {
	Name string
	Data []byte
}

// xorStream XORs b in place with a deterministic xorshift32 keystream. It is its
// own inverse, so packing and unpacking call the same function.
func xorStream(b []byte) {
	s := obfSeed
	for i := range b {
		s ^= s << 13
		s ^= s >> 17
		s ^= s << 5
		b[i] ^= byte(s)
	}
}

// Pack serialises entries into an obfuscated pak blob.
func Pack(entries []Entry) []byte {
	var payload []byte
	var tmp [binary.MaxVarintLen64]byte
	putUvarint := func(v uint64) {
		n := binary.PutUvarint(tmp[:], v)
		payload = append(payload, tmp[:n]...)
	}

	putUvarint(uint64(len(entries)))
	for _, e := range entries {
		putUvarint(uint64(len(e.Name)))
		payload = append(payload, e.Name...)
		putUvarint(uint64(len(e.Data)))
		payload = append(payload, e.Data...)
	}

	xorStream(payload)

	out := make([]byte, 0, len(magic)+len(payload))
	out = append(out, magic...)
	out = append(out, payload...)
	return out
}

// Unpack reverses Pack, returning a map of entry name to its original bytes.
func Unpack(raw []byte) (map[string][]byte, error) {
	if len(raw) < len(magic) || !bytes.Equal(raw[:len(magic)], magic) {
		return nil, errors.New("sndpak: bad magic (not a pak file)")
	}

	payload := make([]byte, len(raw)-len(magic))
	copy(payload, raw[len(magic):])
	xorStream(payload)

	off := 0
	readUvarint := func() (uint64, error) {
		v, n := binary.Uvarint(payload[off:])
		if n <= 0 {
			return 0, errors.New("sndpak: corrupt varint")
		}
		off += n
		return v, nil
	}
	readBytes := func(n uint64) ([]byte, error) {
		if uint64(off)+n > uint64(len(payload)) {
			return nil, fmt.Errorf("sndpak: truncated (want %d bytes at %d of %d)", n, off, len(payload))
		}
		b := payload[off : off+int(n)]
		off += int(n)
		return b, nil
	}

	count, err := readUvarint()
	if err != nil {
		return nil, err
	}
	out := make(map[string][]byte, count)
	for i := uint64(0); i < count; i++ {
		nameLen, err := readUvarint()
		if err != nil {
			return nil, err
		}
		name, err := readBytes(nameLen)
		if err != nil {
			return nil, err
		}
		dataLen, err := readUvarint()
		if err != nil {
			return nil, err
		}
		data, err := readBytes(dataLen)
		if err != nil {
			return nil, err
		}
		// Copy out so the returned bytes don't alias the scratch payload.
		blob := make([]byte, len(data))
		copy(blob, data)
		out[string(name)] = blob
	}
	return out, nil
}
