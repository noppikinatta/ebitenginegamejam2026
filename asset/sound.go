// Copyright 2022 noppikinatta
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package asset

import (
	"bytes"
	_ "embed"
	"errors"
	"io"
	"log"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

//go:embed sound/bgm.wav
var bgm []byte

//go:embed sound/fire.wav
var seFire []byte

//go:embed sound/explosion.wav
var seExplosion []byte

//go:embed sound/hit.wav
var seHit []byte

const sampleRate int = 48000

var context *audio.Context

func init() {
	context = audio.NewContext(sampleRate)
	seBytes = map[Sound][]byte{}
}

// Sound identifies a loadable sound. BGM loops; the SE* values are one-shot
// effects that may overlap (each Play spins up a fresh, cheap player).
type Sound int

const (
	BGM Sound = iota
	SEFire
	SEExplosion
	SEPlayerHit
)

type fileType int

const (
	fileTypeWav fileType = iota
	fileTypeMp3
	fileTypeOgg
)

// soundSpec describes how to load one sound.
type soundSpec struct {
	resource []byte
	sound    Sound
	fileType fileType
	volume   float64
}

var soundSpecs = []soundSpec{
	{seFire, SEFire, fileTypeWav, 0.2},
	{seExplosion, SEExplosion, fileTypeWav, 0.3},
	{seHit, SEPlayerHit, fileTypeWav, 0.4},
	{bgm, BGM, fileTypeWav, 0.25},
}

// seBytes holds the decoded PCM for each one-shot effect so PlaySound can create
// a fresh player per play (allowing the same SE to overlap). bgmPlayer is the
// single looping player reused via Rewind.
var (
	seBytes   map[Sound][]byte
	seVolume  = map[Sound]float64{}
	bgmPlayer *audio.Player
)

// LoadSounds decodes the embedded sounds. A failed track is logged and skipped
// so a single bad asset does not take down the whole game (placeholder audio
// still decodes, but real assets can be swapped in freely).
func LoadSounds() error {
	for _, s := range soundSpecs {
		if err := load(s); err != nil {
			log.Printf("skip loading sound %d: %v", s.sound, err)
			continue
		}
	}
	return nil
}

func decode(resource []byte, ftype fileType) (io.ReadSeeker, error) {
	switch ftype {
	case fileTypeWav:
		return wav.DecodeWithSampleRate(sampleRate, bytes.NewReader(resource))
	case fileTypeMp3:
		return mp3.DecodeWithSampleRate(sampleRate, bytes.NewReader(resource))
	case fileTypeOgg:
		return vorbis.DecodeWithSampleRate(sampleRate, bytes.NewReader(resource))
	default:
		return nil, errors.New("not supported filetype")
	}
}

func load(s soundSpec) error {
	stream, err := decode(s.resource, s.fileType)
	if err != nil {
		return err
	}

	if s.sound == BGM {
		// One reusable looping player; the loop spans the whole decoded stream.
		var loopLen int64
		if l, ok := stream.(interface{ Length() int64 }); ok {
			loopLen = l.Length()
		}
		p, err := context.NewPlayer(audio.NewInfiniteLoop(stream, loopLen))
		if err != nil {
			return err
		}
		p.SetVolume(s.volume)
		bgmPlayer = p
		return nil
	}

	// One-shot SE: keep the decoded PCM so each play gets its own player.
	pcm, err := io.ReadAll(stream)
	if err != nil {
		return err
	}
	seBytes[s.sound] = pcm
	seVolume[s.sound] = s.volume
	return nil
}

// PlaySound plays a sound. BGM (re)starts the looping track; SE* play a fresh
// overlapping one-shot. Unloaded sounds are a no-op.
func PlaySound(s Sound) {
	if s == BGM {
		PlayBGM()
		return
	}
	pcm := seBytes[s]
	if pcm == nil {
		return
	}
	p := context.NewPlayerFromBytes(pcm)
	p.SetVolume(seVolume[s])
	p.Play() // GC'd once playback finishes
}

// PlayBGM starts (or restarts) the looping background track.
func PlayBGM() {
	if bgmPlayer == nil {
		return
	}
	if err := bgmPlayer.Rewind(); err != nil {
		log.Println(err)
	}
	bgmPlayer.Play()
}

// StopBGM pauses the background track.
func StopBGM() {
	if bgmPlayer == nil {
		return
	}
	bgmPlayer.Pause()
}
