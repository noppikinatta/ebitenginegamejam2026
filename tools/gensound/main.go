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

// waveform maps a phase measured in cycles (not radians) to amplitude -1..1.
// Phase need not be wrapped; each waveform takes its own fractional part, so a
// running cycle count from a pitch sweep can be passed straight in.
type waveform func(cycles float64) float64

func wSine(c float64) float64 { return math.Sin(2 * math.Pi * c) }

// wSquare is a hard square wave: full +1 for the first half of each cycle, -1
// for the second. Punchy and hollow — good for retro blips and booms.
func wSquare(c float64) float64 {
	if c-math.Floor(c) < 0.5 {
		return 1
	}
	return -1
}

// wSaw is a rising sawtooth ramp from -1 to +1 each cycle. Bright and buzzy —
// the classic "laser"/engine timbre.
func wSaw(c float64) float64 {
	f := c - math.Floor(c)
	return 2*f - 1
}

// wTri is a triangle wave: -1→+1 over the first half, +1→-1 over the second.
// Softer than square/saw (fewer high harmonics) — a mellow, woody tone.
func wTri(c float64) float64 {
	f := c - math.Floor(c)
	if f < 0.5 {
		return 4*f - 1
	}
	return 3 - 4*f
}

// env is the shared 5ms-attack / 20ms-release amplitude envelope at time t of a
// dur-second sound, so every generated tone fades in/out without clicks.
func env(t, dur float64) float64 {
	return math.Min(1, math.Min(t/0.005, (dur-t)/0.02))
}

// osc renders a fixed-frequency tone of the given waveform with the shared
// click-free envelope.
func osc(w waveform, freq, dur, vol float64) []float64 {
	s := make([]float64, n(dur))
	for i := range s {
		t := float64(i) / sampleRate
		s[i] = w(freq*t) * env(t, dur) * vol
	}
	return s
}

// oscSweep renders a tone of the given waveform whose pitch glides from f0 to f1
// over dur seconds. Phase is accumulated in cycles so the sweep stays in tune
// for any waveform (sine/square/saw/triangle).
func oscSweep(w waveform, f0, f1, dur, vol float64) []float64 {
	s := make([]float64, n(dur))
	cycles := 0.0
	for i := range s {
		t := float64(i) / sampleRate
		f := f0 + (f1-f0)*(t/dur)
		cycles += f / sampleRate
		s[i] = w(cycles) * env(t, dur) * vol
	}
	return s
}

// tone makes a sine of freq for dur seconds (sine wrapper over osc; used by BGM).
func tone(freq, dur, vol float64) []float64 { return osc(wSine, freq, dur, vol) }

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

// sweep makes a sine that glides from f0 to f1 over dur seconds (sine wrapper).
func sweep(f0, f1, dur, vol float64) []float64 { return oscSweep(wSine, f0, f1, dur, vol) }

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
	// timbre — now built from the square/saw/triangle waveforms as well as sine
	// and noise — so they are easy to tell apart (and easy to swap individually).
	// The file base name must match the pak entry name in asset/sound.go.

	// Cannon: a heavy boom. A square wave dropping in pitch gives a punchy
	// hollow thump, with a noise transient for the muzzle blast.
	cannon := oscSweep(wSquare, 190, 90, 0.11, 0.45)
	mix(cannon, noise(0.06, 0.3))
	writeWAV(out+"/fire_cannon.wav", cannon)

	// Shotgun: a noisy spread blast — mostly noise with a low sawtooth growl.
	shotgun := noise(0.12, 0.6)
	mix(shotgun, osc(wSaw, 130, 0.10, 0.25))
	writeWAV(out+"/fire_shotgun.wav", shotgun)

	// Sniper: a high, sharp crack. A short bright sawtooth cuts through.
	writeWAV(out+"/fire_sniper.wav", osc(wSaw, 1000, 0.05, 0.45))

	// Laser: the classic descending buzzy zap — sawtooth sweeping down.
	writeWAV(out+"/fire_laser.wav", oscSweep(wSaw, 1600, 360, 0.13, 0.4))

	// Gatling: a tiny dry click — a very short square blip.
	writeWAV(out+"/fire_gatling.wav", osc(wSquare, 560, 0.025, 0.4))

	// Grenade: a soft low "thunk" — a mellow triangle dropping in pitch, plus a
	// little noise body.
	grenade := oscSweep(wTri, 180, 80, 0.10, 0.6)
	mix(grenade, noise(0.05, 0.18))
	writeWAV(out+"/fire_grenade.wav", grenade)

	// CIWS: a fast mid blip, brighter and harder than the gatling — square wave.
	writeWAV(out+"/fire_ciws.wav", osc(wSquare, 820, 0.03, 0.4))

	// Missile: a rising whoosh — sawtooth sweeping up with a noise wash.
	missile := oscSweep(wSaw, 240, 760, 0.18, 0.4)
	mix(missile, noise(0.18, 0.2))
	writeWAV(out+"/fire_missile.wav", missile)

	// SE: explosion — a low triangle body dropping in pitch under a big noise
	// burst, for a fuller boom than a plain sine.
	{
		body := oscSweep(wTri, 140, 55, 0.35, 0.6)
		mix(body, noise(0.35, 0.55))
		writeWAV(out+"/explosion.wav", body)
	}

	// SE: player hit — a harsh, attention-grabbing square wave sliding down, so
	// taking damage sounds nastier than the soft sine it replaces.
	writeWAV(out+"/hit.wav", oscSweep(wSquare, 320, 130, 0.16, 0.45))

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
