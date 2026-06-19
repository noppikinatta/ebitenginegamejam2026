//go:build ignore
// +build ignore

package main

import (
	"encoding/binary"
	"math"
	"math/rand"
	"os"
)

const sampleRate = 48000

// writeWAV writes 16-bit PCM mono samples (range -1..1) as a WAV file.
func writeWAV(path string, samples []float64) {
	buf := make([]byte, 0, 44+len(samples)*2)
	put := func(b ...byte) { buf = append(buf, b...) }
	u32 := func(v uint32) { var b [4]byte; binary.LittleEndian.PutUint32(b[:], v); put(b[:]...) }
	u16 := func(v uint16) { var b [2]byte; binary.LittleEndian.PutUint16(b[:], v); put(b[:]...) }

	dataLen := uint32(len(samples) * 2)
	put('R', 'I', 'F', 'F')
	u32(36 + dataLen)
	put('W', 'A', 'V', 'E')
	put('f', 'm', 't', ' ')
	u32(16)                       // fmt chunk size
	u16(1)                        // PCM
	u16(1)                        // mono
	u32(sampleRate)               // sample rate
	u32(sampleRate * 2)           // byte rate (rate * channels * bytesPerSample)
	u16(2)                        // block align
	u16(16)                       // bits per sample
	put('d', 'a', 't', 'a')
	u32(dataLen)
	for _, s := range samples {
		if s > 1 {
			s = 1
		} else if s < -1 {
			s = -1
		}
		var b [2]byte
		binary.LittleEndian.PutUint16(b[:], uint16(int16(s*32767)))
		put(b[:]...)
	}
	if err := os.WriteFile(path, buf, 0644); err != nil {
		panic(err)
	}
}

func n(d float64) int { return int(d * sampleRate) }

// tone makes a sine of freq for dur seconds with a short attack/decay envelope.
func tone(freq, dur, vol float64) []float64 {
	s := make([]float64, n(dur))
	for i := range s {
		t := float64(i) / sampleRate
		env := math.Min(1, math.Min(t/0.005, (dur-t)/0.02)) // 5ms attack, 20ms release
		s[i] = math.Sin(2*math.Pi*freq*t) * env * vol
	}
	return s
}

// noise makes a decaying noise burst (used for explosion).
func noise(dur, vol float64) []float64 {
	s := make([]float64, n(dur))
	r := rand.New(rand.NewSource(1))
	for i := range s {
		t := float64(i) / float64(len(s))
		s[i] = (r.Float64()*2 - 1) * (1 - t) * (1 - t) * vol
	}
	return s
}

func main() {
	out := os.Args[1]

	// SE: weapon fire — short high blip.
	writeWAV(out+"/fire.wav", tone(660, 0.06, 0.6))

	// SE: explosion — low tone mixed with a noise burst.
	{
		body := tone(120, 0.35, 0.7)
		ns := noise(0.35, 0.5)
		for i := range body {
			body[i] += ns[i]
		}
		writeWAV(out+"/explosion.wav", body)
	}

	// SE: player hit — short descending low tone.
	{
		dur := 0.18
		s := make([]float64, n(dur))
		for i := range s {
			t := float64(i) / sampleRate
			f := 300 - 150*(t/dur) // glide down
			env := math.Min(1, (dur-t)/0.03)
			s[i] = math.Sin(2*math.Pi*f*t) * env * 0.6
		}
		writeWAV(out+"/hit.wav", s)
	}

	// BGM: a short seamless 2-second loop (simple arpeggio).
	{
		notes := []float64{220, 277, 330, 277} // A3 C#4 E4 C#4
		var bgm []float64
		for _, f := range notes {
			bgm = append(bgm, tone(f, 0.5, 0.4)...)
		}
		writeWAV(out+"/bgm.wav", bgm)
	}
}
