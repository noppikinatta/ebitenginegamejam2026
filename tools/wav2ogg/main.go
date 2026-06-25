//go:build ignore
// +build ignore

// wav2ogg converts looping BGM WAVs to Ogg Vorbis for a smaller WASM payload,
// while preserving Ebitengine's seamless looping.
//
// # WHY THIS TOOL EXISTS
//
// The game loops BGM with audio.NewInfiniteLoop(stream, loopLen). For a WAV the
// loop length is taken from the decoded stream's Length(), which is exact, so the
// loop is sample-accurate and silent at the seam. Ogg Vorbis is ~10x smaller but
// lossy: its decoded length includes encoder priming/padding, so reading right up
// to the end (or trusting its Length()) clicks. The fix is two-fold:
//
//  1. Record the EXACT loop length the WAV path would have produced (this tool
//     computes it identically to Ebiten's wav.DecodeWithSampleRate + Resampling,
//     so the value matches the runtime to the byte) and pass THAT constant to
//     NewInfiniteLoop instead of the ogg stream's own Length().
//  2. Append ~1s of digital silence before encoding, so the encoder's padding
//     lands inside the silence and the real content region [0,loopLen) stays
//     intact. The loop never reaches the silence (loopLen < ogg length).
//
// IMPORTANT: the ogg keeps the source sample rate and channel count, so the
// runtime resample path (e.g. 44100 -> 44100) is identical for the wav and the
// ogg. That is what makes the recorded loopLen reusable for the ogg.
//
// OUTPUT
//
//	<outDir>/<base>.ogg        the encoded, silence-padded track
//	<outDir>/<base>.ogg.loop   the loop length in bytes (decimal text), for the
//	                           game to read (embed) or to paste into a constant
//
// The loop length is also printed, with a ready-to-paste Go snippet.
//
// REQUIREMENTS: ffmpeg with libvorbis on PATH (encode step only). Pass
// -encode=false to skip encoding and only compute the loop length / write the
// padded WAV (-keep-wav) — handy on a headless box without ffmpeg.
//
// Run: go run tools/wav2ogg/main.go [flags] <outDir> <in1.wav> [in2.wav ...]
package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// contextRate is the game's audio context sample rate (asset.sampleRate). Ebiten
// resamples every decoded stream to this rate, so the loop length is measured in
// 16-bit stereo bytes at this rate.
const contextRate = 44100

func main() {
	silence := flag.Float64("silence", 1.0, "seconds of trailing silence to pad before encoding")
	quality := flag.Int("q", 5, "Ogg Vorbis quality for ffmpeg -q:a (0..10; ~5 is a good BGM default)")
	encode := flag.Bool("encode", true, "run ffmpeg to produce the .ogg (false: only compute loop length)")
	keepWav := flag.Bool("keep-wav", false, "also write the padded <base>.padded.wav next to the output")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: go run tools/wav2ogg/main.go [flags] <outDir> <in1.wav> [in2.wav ...]")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() < 2 {
		flag.Usage()
		os.Exit(2)
	}
	outDir := flag.Arg(0)
	inputs := flag.Args()[1:]

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "mkdir:", err)
		os.Exit(1)
	}
	if *encode {
		if _, err := exec.LookPath("ffmpeg"); err != nil {
			fmt.Fprintln(os.Stderr, "ffmpeg not found on PATH; install it or pass -encode=false")
			os.Exit(1)
		}
	}

	type result struct {
		base    string
		loopLen int64
	}
	var results []result

	for _, in := range inputs {
		w, err := readWAV(in)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", in, err)
			os.Exit(1)
		}
		loopLen := loopLengthBytes(w)
		base := strings.TrimSuffix(filepath.Base(in), filepath.Ext(in))

		srcSec := float64(w.frames) / float64(w.sampleRate)
		fmt.Printf("%s\n", in)
		fmt.Printf("  source : %d Hz, %d ch, %d-bit, %d frames (%.2fs)\n", w.sampleRate, w.channels, w.bitsPerSample, w.frames, srcSec)
		fmt.Printf("  loopLen: %d bytes  (= %d frames @ %d Hz 16-bit stereo, %.3fs)\n", loopLen, loopLen/4, contextRate, float64(loopLen/4)/contextRate)

		// Build the silence-padded WAV (canonical 16-bit PCM, source rate/channels).
		padded := padWithSilence(w, *silence)

		if *keepWav {
			pw := filepath.Join(outDir, base+".padded.wav")
			if err := os.WriteFile(pw, padded, 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "write padded wav:", err)
				os.Exit(1)
			}
			fmt.Printf("  wrote  : %s\n", pw)
		}

		if *encode {
			oggPath := filepath.Join(outDir, base+".ogg")
			if err := encodeOgg(padded, oggPath, w.sampleRate, w.channels, *quality); err != nil {
				fmt.Fprintf(os.Stderr, "encode %s: %v\n", oggPath, err)
				os.Exit(1)
			}
			st, _ := os.Stat(oggPath)
			fmt.Printf("  wrote  : %s (%d bytes)\n", oggPath, st.Size())
		}

		// Record the loop length next to the output for the game to read/embed.
		loopPath := filepath.Join(outDir, base+".ogg.loop")
		if err := os.WriteFile(loopPath, []byte(fmt.Sprintf("%d\n", loopLen)), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "write loop file:", err)
			os.Exit(1)
		}
		fmt.Printf("  wrote  : %s\n", loopPath)

		results = append(results, result{base: base, loopLen: loopLen})
	}

	// Ready-to-paste Go: a map of track base name -> loop length in bytes.
	fmt.Println("\n// Loop lengths (bytes, 44100 Hz 16-bit stereo) for audio.NewInfiniteLoop:")
	fmt.Println("var bgmLoopLen = map[string]int64{")
	for _, r := range results {
		fmt.Printf("\t%q: %d,\n", r.base, r.loopLen)
	}
	fmt.Println("}")
}

// wavData is the parsed, decoded-to-16-bit view of a source WAV.
type wavData struct {
	sampleRate    int
	channels      int
	bitsPerSample int
	frames        int64  // samples per channel
	pcm           []byte // raw data-chunk bytes, source format
}

// readWAV parses a PCM WAV (RIFF), skipping non-essential chunks (JUNK, LIST, ...).
func readWAV(path string) (*wavData, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) < 12 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return nil, errors.New("not a RIFF/WAVE file")
	}
	var (
		haveFmt                     bool
		audioFormat, channels, bits int
		sampleRate                  int
		dataOff, dataSize           int
	)
	i := 12
	for i+8 <= len(b) {
		id := string(b[i : i+4])
		sz := int(binary.LittleEndian.Uint32(b[i+4 : i+8]))
		body := i + 8
		if body+sz > len(b) {
			sz = len(b) - body // tolerate a truncated final chunk
		}
		switch id {
		case "fmt ":
			if sz < 16 {
				return nil, errors.New("short fmt chunk")
			}
			audioFormat = int(binary.LittleEndian.Uint16(b[body : body+2]))
			channels = int(binary.LittleEndian.Uint16(b[body+2 : body+4]))
			sampleRate = int(binary.LittleEndian.Uint32(b[body+4 : body+8]))
			bits = int(binary.LittleEndian.Uint16(b[body+14 : body+16]))
			haveFmt = true
		case "data":
			dataOff, dataSize = body, sz
		}
		i = body + sz
		if sz%2 == 1 {
			i++ // chunks are word-aligned
		}
		if dataOff != 0 && haveFmt {
			break
		}
	}
	if !haveFmt {
		return nil, errors.New("no fmt chunk")
	}
	if dataOff == 0 {
		return nil, errors.New("no data chunk")
	}
	if audioFormat != 1 {
		return nil, fmt.Errorf("unsupported audio format %d (only PCM is supported, like Ebiten's wav decoder)", audioFormat)
	}
	if bits != 8 && bits != 16 {
		return nil, fmt.Errorf("unsupported bit depth %d (Ebiten's wav decoder handles 8 or 16)", bits)
	}
	if channels != 1 && channels != 2 {
		return nil, fmt.Errorf("unsupported channel count %d (1 or 2)", channels)
	}
	frames := int64(dataSize) / int64(channels*bits/8)
	return &wavData{
		sampleRate:    sampleRate,
		channels:      channels,
		bitsPerSample: bits,
		frames:        frames,
		pcm:           b[dataOff : dataOff+dataSize],
	}, nil
}

// loopLengthBytes returns the loop length in bytes EXACTLY as the game computes
// it at runtime: Ebiten's wav.DecodeWithSampleRate converts the source to 16-bit
// stereo (so the decoded size is frames*4), then, if the source rate differs from
// the context rate, wraps it in convert.Resampling whose Length() is
// int64(size * to / from) aligned down to a 4-byte (stereo 16-bit) frame.
func loopLengthBytes(w *wavData) int64 {
	const decodedBytesPerFrame = 4 // 2 channels * 2 bytes (Ebiten upmixes mono and widens 8-bit)
	decodedSize := w.frames * decodedBytesPerFrame
	if w.sampleRate == contextRate {
		return decodedSize
	}
	// Mirror convert.Resampling.Length() precisely, including float64 order.
	s := int64(float64(decodedSize) * float64(contextRate) / float64(w.sampleRate))
	return s / decodedBytesPerFrame * decodedBytesPerFrame
}

// padWithSilence returns a canonical 16-bit-or-source-depth PCM WAV (source rate
// and channel count) holding the original samples followed by `sec` seconds of
// digital silence. Non-essential chunks from the source are dropped.
func padWithSilence(w *wavData, sec float64) []byte {
	bytesPerFrame := w.channels * w.bitsPerSample / 8
	silenceFrames := int64(sec*float64(w.sampleRate) + 0.5)
	silenceBytes := silenceFrames * int64(bytesPerFrame)

	dataLen := int64(len(w.pcm)) + silenceBytes
	out := make([]byte, 0, 44+dataLen)
	put := func(b ...byte) { out = append(out, b...) }
	u16 := func(v uint16) { var t [2]byte; binary.LittleEndian.PutUint16(t[:], v); put(t[:]...) }
	u32 := func(v uint32) { var t [4]byte; binary.LittleEndian.PutUint32(t[:], v); put(t[:]...) }

	put('R', 'I', 'F', 'F')
	u32(uint32(36 + dataLen))
	put('W', 'A', 'V', 'E')
	put('f', 'm', 't', ' ')
	u32(16)
	u16(1) // PCM
	u16(uint16(w.channels))
	u32(uint32(w.sampleRate))
	u32(uint32(w.sampleRate * bytesPerFrame)) // byte rate
	u16(uint16(bytesPerFrame))                // block align
	u16(uint16(w.bitsPerSample))
	put('d', 'a', 't', 'a')
	u32(uint32(dataLen))
	out = append(out, w.pcm...)
	out = append(out, make([]byte, silenceBytes)...) // zeros = silence
	return out
}

// encodeOgg pipes the padded WAV through ffmpeg's libvorbis encoder, forcing the
// source rate/channels so the runtime resample path matches the WAV's.
func encodeOgg(wavBytes []byte, outPath string, rate, channels, quality int) error {
	cmd := exec.Command("ffmpeg",
		"-y", "-hide_banner", "-loglevel", "error",
		"-f", "wav", "-i", "pipe:0",
		"-c:a", "libvorbis",
		"-q:a", fmt.Sprintf("%d", quality),
		"-ar", fmt.Sprintf("%d", rate),
		"-ac", fmt.Sprintf("%d", channels),
		outPath,
	)
	cmd.Stdin = strings.NewReader(string(wavBytes))
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
