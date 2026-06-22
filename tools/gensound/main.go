//go:build ignore
// +build ignore

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
)

const sampleRate = 48000

// force, when true, overwrites existing files; otherwise writeWAV skips any file
// that already exists so real audio is never clobbered.
var force bool

// writeWAV writes 16-bit PCM mono samples (range -1..1) as a WAV file. Existing
// files are skipped unless -force is set.
func writeWAV(path string, samples []float64) {
	if !force {
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("skip %s (exists; -force to overwrite)\n", path)
			return
		}
	}
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

// sweep makes a sine that glides from f0 to f1 over dur seconds (zap/whoosh).
func sweep(f0, f1, dur, vol float64) []float64 {
	s := make([]float64, n(dur))
	phase := 0.0
	for i := range s {
		t := float64(i) / sampleRate
		f := f0 + (f1-f0)*(t/dur)
		phase += 2 * math.Pi * f / sampleRate
		env := math.Min(1, math.Min(t/0.005, (dur-t)/0.02))
		s[i] = math.Sin(phase) * env * vol
	}
	return s
}

// mix adds src into dst in place (dst length wins).
func mix(dst, src []float64) {
	for i := range dst {
		if i < len(src) {
			dst[i] += src[i]
		}
	}
}

func main() {
	// arg1: directory for SE placeholders (e.g. asset/sound/raw, gitignored).
	// arg2: directory for the BGM placeholder (e.g. asset/sound, committed).
	// BGM is self-authored and lives outside the pak, so it is written separately.
	flag.BoolVar(&force, "force", false, "overwrite existing files (default: skip files that already exist)")
	flag.Parse()

	seDir := flag.Arg(0)
	bgmDir := seDir
	if flag.NArg() > 1 {
		bgmDir = flag.Arg(1)
	}
	out := seDir

	// SE: per-weapon fire. Each weapon gets a deliberately distinct placeholder
	// timbre so they are easy to tell apart (and easy to swap individually). The
	// file base name must match the pak entry name in asset/sound.go (fire_<kind>).
	cannon := tone(200, 0.09, 0.6)
	mix(cannon, noise(0.05, 0.2))
	writeWAV(out+"/fire_cannon.wav", cannon) // low boom

	shotgun := noise(0.10, 0.6)
	mix(shotgun, tone(160, 0.10, 0.3))
	writeWAV(out+"/fire_shotgun.wav", shotgun) // noisy blast

	writeWAV(out+"/fire_sniper.wav", tone(950, 0.05, 0.6))      // high sharp crack
	writeWAV(out+"/fire_laser.wav", sweep(1300, 320, 0.13, 0.5)) // descending zap
	writeWAV(out+"/fire_gatling.wav", tone(520, 0.03, 0.5))     // tiny click
	writeWAV(out+"/fire_grenade.wav", tone(150, 0.08, 0.6))     // low thunk
	writeWAV(out+"/fire_ciws.wav", tone(740, 0.035, 0.5))       // mid blip

	missile := sweep(280, 720, 0.18, 0.5)
	mix(missile, noise(0.18, 0.25))
	writeWAV(out+"/fire_missile.wav", missile) // rising whoosh

	// SE: explosion — low tone mixed with a noise burst.
	{
		body := tone(120, 0.35, 0.7)
		mix(body, noise(0.35, 0.5))
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

	// BGM (two distinct loops). Title = calm arpeggio; Game = faster, driving.
	{
		notes := []float64{220, 277, 330, 277} // A3 C#4 E4 C#4
		var bgm []float64
		for _, f := range notes {
			bgm = append(bgm, tone(f, 0.5, 0.4)...)
		}
		writeWAV(bgmDir+"/bgm_title.wav", bgm)
	}
	{
		notes := []float64{165, 196, 247, 330, 247, 196} // E3 G3 B3 E4 B3 G3
		var bgm []float64
		for _, f := range notes {
			seg := tone(f, 0.28, 0.35)
			mix(seg, tone(f/2, 0.28, 0.2)) // octave-down bass
			bgm = append(bgm, seg...)
		}
		writeWAV(bgmDir+"/bgm_game.wav", bgm)
	}
}
